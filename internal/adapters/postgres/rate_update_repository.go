package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"fxrates/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RateUpdateRepository struct {
	pool *pgxpool.Pool
}

func (r *RateUpdateRepository) ScheduleNewOrGetExisting(ctx context.Context, base string, quote string) (uuid.UUID, error) {
	const q = `
		-- 1) ensure pair exists and get its id
		with pair as (
		  insert into fx_pairs(base, quote) values ($1,$2)
		  on conflict (base, quote) do update
		    set base = excluded.base   -- no-op, just to return id
		  returning id
		)
        -- 2) insert pending update or fetch existing update_id
        insert into fx_rate_updates (pair_id, update_id, status, updated_at)
        select p.id, $3, 'pending', now() from pair p
		on conflict (pair_id) where status = 'pending'
		do update set updated_at = fx_rate_updates.updated_at
        returning update_id;
	`

	var updateID uuid.UUID
	err := r.pool.QueryRow(ctx, q, base, quote, uuid.New()).Scan(&updateID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to ensure an update for '%s/%s': %w", base, quote, err)
	}
	return updateID, nil
}

func (r *RateUpdateRepository) GetPending(ctx context.Context) ([]domain.PendingRateUpdate, error) {
	const q = `
		select fru.update_id, fru.pair_id, fp.base, fp.quote
		from fx_rate_updates fru join fx_pairs fp on fp.id = fru.pair_id
		where fru.status = 'pending';
	`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending rates: %w", err)
	}
	defer rows.Close()

	pending := make([]domain.PendingRateUpdate, 0, 64)
	for rows.Next() {
		var pr domain.PendingRateUpdate
		if err = rows.Scan(&pr.UpdateID, &pr.PairID, &pr.Base, &pr.Quote); err != nil {
			return nil, fmt.Errorf("failed to scan pending rate: %w", err)
		}
		pending = append(pending, pr)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending rates: %w", err)
	}
	return pending, nil
}

func (r *RateUpdateRepository) ApplyUpdates(ctx context.Context, applied []domain.AppliedRateUpdate) error {
	if len(applied) == 0 {
		return nil
	}

	payloadJSON, err := json.Marshal(applied)
	if err != nil {
		return fmt.Errorf("failed to marshal applied rates: %w", err)
	}

	const q = `
		with
		
		-- step 1: parsing input
		input_rows as (select * from json_to_recordset($1::json) as r(pair_id bigint, update_id uuid, value numeric)),
		
		-- step 2: updating fx_rate_updates records and get updated
		update_fru as (
		  update fx_rate_updates fru
		  set value = ir.value, updated_at = now(), status = 'applied'
		  from input_rows ir 
		  where fru.update_id = ir.update_id
		  returning fru.pair_id, fru.value
		)
		
		-- step 3: updating fx_last_rates records
		insert into fx_last_rates(pair_id, value, updated_at)
		select pair_id, value, now() from update_fru
		on conflict (pair_id) do update
		set value = excluded.value, updated_at = now();
	`

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, q, json.RawMessage(payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func NewRateUpdateRepository(pool *pgxpool.Pool) *RateUpdateRepository {
	return &RateUpdateRepository{pool: pool}
}
