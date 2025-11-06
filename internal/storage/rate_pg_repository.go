package storage

import (
	"context"
	"fxrates/internal/domain"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RatePgRepository struct { // TODO: через конструктор
	pool *pgxpool.Pool
}

func (r *RatePgRepository) UpdateRateByCode(ctx context.Context, code string) (uuid.UUID, error) {
	updateID := uuid.New()
	_, err := r.pool.Exec(ctx, `insert into tasks (code, status, update_id) values ($1, $2, $3)`, code, "NEW", updateID)
	if err != nil {
		return uuid.Nil, err
	}
	return updateID, nil
}

type RateInfo struct {
	UpdateID   uuid.UUID
	Code       string
	NewPrice   float64
	UpdateDate time.Time
}

func (r *RatePgRepository) GetRateInfoByUpdateID(updateID uuid.UUID) (*RateInfo, error) {
	sql := `select code, new_price, update_date from tasks where update_id = $1`

	var rateInfo RateInfo

	err := r.pool.QueryRow(context.Background(), sql, updateID).Scan(&rateInfo.Code, &rateInfo.NewPrice, &rateInfo.UpdateDate)
	if err != nil {
		return nil, err
	}

	rateInfo.UpdateID = updateID
	return &rateInfo, nil
}

func (r *RatePgRepository) GetRateByCode(code string) (*domain.Rate, error) {
	sql := `select id, name, code, price, create_date, update_date from rates where code = $1`

	var rate domain.Rate

	err := r.pool.QueryRow(context.Background(), sql, code).Scan(
		&rate.ID,
		&rate.Name,
		&rate.Code,
		&rate.Price,
		&rate.CreateDate,
		&rate.UpdateDate,
	)
	if err != nil {
		return nil, err
	}

	return &rate, nil
}
