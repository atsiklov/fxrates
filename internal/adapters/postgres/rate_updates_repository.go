package postgres

import (
	"context"
	"encoding/json"
	"fxrates/internal/domain"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RateUpdatesRepository struct {
	pool *pgxpool.Pool
}

func (r *RateUpdatesRepository) GetPending(ctx context.Context) ([]domain.PendingRate, error) {
	const q = `
		select fru.pair_id, fp.base, fp.quote
		from fx_rate_updates fru join fx_pairs fp on fp.id = fru.pair_id
		where fru.status = 'pending';
	`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rates := make([]domain.PendingRate, 0, 64)
	for rows.Next() {
		var rate domain.PendingRate
		if err = rows.Scan(&rate.PairID, &rate.Base, &rate.Quote); err != nil {
			return nil, err
		}
		rates = append(rates, rate)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return rates, nil
}

type batchRow struct {
	PairID    int64     `json:"pair_id"`
	Value     float64   `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (r *RateUpdatesRepository) SaveApplied(ctx context.Context, rates []domain.AppliedRate) error {
	if len(rates) == 0 {
		return nil
	}
	payload := make([]batchRow, 0, len(rates))
	for _, rate := range rates {
		payload = append(payload, batchRow{rate.PairID, rate.Value, rate.UpdatedAt})
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	const q = `
		with
		
		-- step 1: parsing input
		input_rows as (select * from json_to_recordset($1::json) as r(pair_id bigint, value numeric)),
		
		-- step 2: updating fx_rate_updates records and get updated
		update_fru as (
		  update fx_rate_updates fru
		  set value = ir.value, updated_at = now(), status = 'applied'
		  from input_rows ir
		  where fru.pair_id = ir.pair_id and fru.status = 'pending'
		)
		
		-- step 3: updating fx_last_rates records
		insert into fx_last_rates(pair_id, value, updated_at)
		select ir.pair_id, ir.value, now()
		from input_rows ir
		on conflict (pair_id) do update
		  set value = excluded.value, updated_at = now();
	`

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, q, json.RawMessage(payloadJSON))
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func NewPostgresRateUpdatesRepository(pool *pgxpool.Pool) *RateUpdatesRepository {
	return &RateUpdatesRepository{pool: pool}
}
