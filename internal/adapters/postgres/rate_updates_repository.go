package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"fxrates/internal/domain"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RateUpdatesRepository struct {
	pool *pgxpool.Pool
}

func (r *RateUpdatesRepository) ScheduleNewOrGetExisting(ctx context.Context, base string, quote string) (uuid.UUID, error) {
	const q = `
		with 
	
		-- 1) ensure pair exists
		ins_pair as (
			insert into fx_pairs(base, quote) values ($1, $2)
			on conflict (base, quote) do nothing
			returning id
		),
		-- 2) get pair id (in case nothing was inserted on 1 step)
		pair as (
			select id from fx_pairs where base = $1 and quote = $2
		),
		-- 3) check if such pair update with status 'pending' already exists
		existing as (
			select fru.update_id from fx_rate_updates fru join pair p on fru.pair_id = p.id
			where fru.status = 'pending' limit 1
		),
		-- 4) if doesn't exist -> create new record
		ins as (
			insert into fx_rate_updates (pair_id, update_id, status, updated_at)
			select p.id, $3, 'pending', now() from pair p
			where not exists (select 1 from existing)
			returning update_id
		)
		-- 5) returning either existing update_id or new one. Never both - impossible
		select update_id from existing
		union all
		select update_id from ins
		limit 1;
	`

	var updateID uuid.UUID
	err := r.pool.QueryRow(ctx, q, base, quote, uuid.New()).Scan(&updateID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to ensure an update for '%q/%q: %w", base, quote, err)
	}
	return updateID, nil
}

func (r *RateUpdatesRepository) GetByUpdateID(ctx context.Context, updateID uuid.UUID) (*domain.AppliedRate, error) {
	const q = `
        select fp.id, fp.base, fp.quote, coalesce(fru.value, 0.0) as value, fru.updated_at, fru.status
        from fx_rate_updates fru join fx_pairs fp on fru.pair_id = fp.id
        where fru.update_id = $1;
    `

	var rate domain.AppliedRate
	var status string

	if err := r.pool.QueryRow(ctx, q, updateID).Scan(
		&rate.PairID,
		&rate.Base,
		&rate.Quote,
		&rate.Value,
		&rate.UpdatedAt,
		// -----
		&status,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRateNotFound
		}
		return nil, fmt.Errorf("failed to select rate for update ID %q: %w", updateID, err)
	}
	if status != "applied" {
		return nil, domain.ErrRateNotApplied
	}
	return &rate, nil
}

func (r *RateUpdatesRepository) GetPending(ctx context.Context) ([]domain.PendingRate, error) {
	const q = `
		select fru.pair_id, fp.base, fp.quote
		from fx_rate_updates fru join fx_pairs fp on fp.id = fru.pair_id
		where fru.status = 'pending';
	`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending rates: %w", err)
	}
	defer rows.Close()

	pending := make([]domain.PendingRate, 0, 64)
	for rows.Next() {
		var pr domain.PendingRate
		if err = rows.Scan(&pr.PairID, &pr.Base, &pr.Quote); err != nil {
			return nil, fmt.Errorf("failed to scan pending rate: %w", err)
		}
		pending = append(pending, pr)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending rates: %w", err)
	}
	return pending, nil
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
		return fmt.Errorf("failed to marshal applied rates: %w", err)
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
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, q, json.RawMessage(payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func NewRateUpdatesRepository(pool *pgxpool.Pool) *RateUpdatesRepository {
	return &RateUpdatesRepository{pool: pool}
}
