package postgres

import (
	"context"
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
	updateID := uuid.New()

	const q = `
        insert into fx_rate_updates (pair_id, update_id, status, updated_at)
        select fp.id, $2, 'pending', now()
        from fx_pairs fp where fp.base = $1 and fp.quote = $2;
    `

	tag, err := r.pool.Exec(ctx, q, base, quote, updateID)
	if err != nil {
		return uuid.Nil, err
	}
	if tag.RowsAffected() == 0 {
		return uuid.Nil, pgx.ErrNoRows
	}
	return updateID, nil
}

// GetByUpdateID returns the applied rate for a particular update.
func (r *RateRepository) GetByUpdateID(ctx context.Context, updateID uuid.UUID) (*domain.AppliedRate, error) {
	const q = `
        select fp.id, fp.base, fp.quote, fru.value, fru.updated_at
        from fx_rate_updates fru
        join fx_pairs fp on fru.pair_id = fp.id
        where fru.update_id = $1;
    `

	var rate domain.AppliedRate
	if err := r.pool.QueryRow(ctx, q, updateID).Scan(
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
