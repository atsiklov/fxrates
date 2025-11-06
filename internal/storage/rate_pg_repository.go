package storage

import (
	"fxrates/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RatePgRepository struct { // TODO: через конструктор
	pool *pgxpool.Pool
}

func (*RatePgRepository) UpdateRate() error {
	// TODO: ...
	return nil
}

func (*RatePgRepository) GetRateByRefreshID() (*domain.Rate, error) {
	// TODO: ...
	return &domain.Rate{ID: "id", Name: "name"}, nil
}

func (*RatePgRepository) GetRateByCode(code string) (*domain.Rate, error) {
	// TODO: ...
	return &domain.Rate{ID: "id", Name: "name"}, nil
}
