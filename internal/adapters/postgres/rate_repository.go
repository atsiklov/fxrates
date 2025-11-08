package postgres

import (
	"context"
	"errors"
	"fxrates/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RateRepository struct {
	pool *pgxpool.Pool
}

// UpdateByCode schedules refresh for the first pair that matches provided base code.
func (r *RateRepository) UpdateByCode(ctx context.Context, base string, quote string) (uuid.UUID, error) {
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
		-- 3) check if such pair update with 'pending' status already exists
		existing as (
			select fru.update_id from fx_rate_updates fru join pair p on fru.pair_id = p.id
			where fru.status = 'pending' limit 1
		),
		-- 4) if doesn't exist -> creating new record
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
		return uuid.Nil, err // todo: add custom error
	}
	return updateID, nil
}

// GetByUpdateID returns the applied rate for a particular update.
func (r *RateRepository) GetByUpdateID(ctx context.Context, updateID uuid.UUID) (*domain.AppliedRate, error) {
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
			return nil, errors.New("not found") // todo: add custom error
		}
		return nil, err
	}

	if status != "applied" {
		return nil, errors.New("not updated yet") // todo: add custom error
	}

	return &rate, nil
}

// GetByCode returns the last stored rate for the specified base currency.
func (r *RateRepository) GetByCode(ctx context.Context, base string, quote string) (*domain.AppliedRate, error) {
	const q = `
        select fp.id, fp.base, fp.quote, flr.value, flr.updated_at
        from fx_last_rates flr join fx_pairs fp on flr.pair_id = fp.id
        where fp.base = $1 and fp.quote = $2;
    `

	var rate domain.AppliedRate
	if err := r.pool.QueryRow(ctx, q, base, quote).Scan(
		&rate.PairID,
		&rate.Base,
		&rate.Quote,
		&rate.Value,
		&rate.UpdatedAt,
	); err != nil {
		return nil, err
	}

	return &rate, nil
}

func NewPostgresRateRepository(pool *pgxpool.Pool) *RateRepository {
	return &RateRepository{pool: pool}
}
