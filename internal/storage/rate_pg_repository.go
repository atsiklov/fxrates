package storage

import (
	"context"
	"fxrates/internal/domain"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ratePgRepository struct {
	pool *pgxpool.Pool
}

func (r *ratePgRepository) UpdateRateByCode(ctx context.Context, code string) (uuid.UUID, error) {
	updateID := uuid.New()
	_, err := r.pool.Exec(ctx, `insert into tasks (code, status, update_id) values ($1, $2, $3)`, code, "NEW", updateID)
	if err != nil {
		return uuid.Nil, err
	}
	return updateID, nil
}

type RateInfo struct {
	UpdateID  uuid.UUID
	Code      string
	NewPrice  float64
	UpdatedAt *time.Time
}

func (r *ratePgRepository) GetRateInfoByUpdateID(ctx context.Context, updateID uuid.UUID) (*RateInfo, error) {
	sql := `select code, new_price, updated_at from tasks where update_id = $1`

	var rateInfo RateInfo

	err := r.pool.QueryRow(ctx, sql, updateID).Scan(&rateInfo.Code, &rateInfo.NewPrice, &rateInfo.UpdatedAt)
	if err != nil {
		return nil, err
	}

	rateInfo.UpdateID = updateID
	return &rateInfo, nil
}

func (r *ratePgRepository) GetRateByCode(ctx context.Context, code string) (*domain.Rate, error) {
	sql := `select id, name, code, price, created_at, updated_at from rates where code = $1`

	var rate domain.Rate

	err := r.pool.QueryRow(ctx, sql, code).Scan(
		&rate.ID,
		&rate.Name,
		&rate.Code,
		&rate.Price,
		&rate.CreatedAt,
		&rate.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &rate, nil
}

func NewRatePgRepository(pool *pgxpool.Pool) RateRepository {
	return &ratePgRepository{pool: pool}
}
