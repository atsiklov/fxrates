package postgres_test

import (
	"context"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	"fxrates/internal/adapters/postgres"
	"fxrates/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const migrationsDir = "../../platform/db/migrations"

var (
	pgSetupOnce sync.Once

	pgContainer *tcpg.PostgresContainer
	pgConnStr   string
)

func TestMain(m *testing.M) {
	code := m.Run()
	if pgContainer != nil {
		_ = pgContainer.Terminate(context.Background())
	}
	os.Exit(code)
}

func setupPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()

	pgSetupOnce.Do(func() {
		startPostgres(t)
	})

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, pgConnStr)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	require.NoError(t, resetDatabase(ctx, pool))

	return pool
}

func startPostgres(t *testing.T) {
	ctx := context.Background()
	pg, err := tcpg.Run(ctx,
		"postgres:16-alpine",
		tcpg.WithDatabase("postgres"),
		tcpg.WithUsername("postgres"),
		tcpg.WithPassword("postgres"),
	)
	require.NoError(t, err)

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := goose.OpenDBWithDriver("pgx", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	require.Eventually(t, func() bool {
		pingCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		return db.PingContext(pingCtx) == nil
	}, 15*time.Second, 500*time.Millisecond)

	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.UpContext(ctx, db, migrationsDir))

	pgContainer = pg
	pgConnStr = dsn
}

func resetDatabase(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `truncate table fx_rate_updates, fx_last_rates, fx_pairs, currencies restart identity cascade`); err != nil {
		return err
	}
	return nil
}

// ---------- RateRepository tests ----------

func TestRateRepository_GetByCodes_NotFound(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateRepository(pool)

	ctx := context.Background()
	_, err := repo.GetByCodes(ctx, "USD", "EUR")
	require.ErrorIs(t, err, domain.ErrRateNotFound)
}

func TestRateRepository_GetByCodes_Success(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `insert into currencies(code) values ('USD'),('EUR')`)
	require.NoError(t, err)

	// Insert pair and last rate.
	var pairID int64
	err = pool.QueryRow(ctx, `insert into fx_pairs(base, quote) values($1,$2) returning id`, "USD", "EUR").Scan(&pairID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `insert into fx_last_rates(pair_id, value, updated_at) values ($1, $2, now())`, pairID, 1.23456)
	require.NoError(t, err)

	rate, err := repo.GetByCodes(ctx, "USD", "EUR")
	require.NoError(t, err)
	require.Equal(t, pairID, rate.PairID)
	require.Equal(t, "USD", rate.Base)
	require.Equal(t, "EUR", rate.Quote)
	require.InDelta(t, 1.2346, rate.Value, 0.00001) // rounded to 4 decimals
	require.False(t, rate.UpdatedAt.IsZero())
}

func TestRateRepository_GetByCodes_DBError(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateRepository(pool)

	// Use a canceled context to force an error path distinct from ErrRateNotFound.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := repo.GetByCodes(ctx, "USD", "EUR")
	require.Error(t, err)
	require.NotErrorIs(t, err, domain.ErrRateNotFound)
}

func TestRateRepository_GetByUpdateID_NotFound(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateRepository(pool)

	ctx := context.Background()
	_, _, err := repo.GetByUpdateID(ctx, uuid.New())
	require.ErrorIs(t, err, domain.ErrRateNotFound)
}

func TestRateRepository_GetByUpdateID_Pending_NoValue(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `insert into currencies(code) values ('USD'),('JPY')`)
	require.NoError(t, err)

	var pairID int64
	err = pool.QueryRow(ctx, `insert into fx_pairs(base, quote) values($1,$2) returning id`, "USD", "JPY").Scan(&pairID)
	require.NoError(t, err)
	updID := uuid.New()
	_, err = pool.Exec(ctx, `insert into fx_rate_updates(pair_id, update_id, status) values ($1,$2,'pending')`, pairID, updID)
	require.NoError(t, err)

	rate, status, err := repo.GetByUpdateID(ctx, updID)
	require.NoError(t, err)
	require.Equal(t, pairID, rate.PairID)
	require.Equal(t, domain.StatusPending, status)
	require.Equal(t, -1.0, rate.Value) // explicitly set bad value when pending
}

func TestRateRepository_GetByUpdateID_Applied_WithValue(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `insert into currencies(code) values ('GBP'),('USD')`)
	require.NoError(t, err)

	var pairID int64
	err = pool.QueryRow(ctx, `insert into fx_pairs(base, quote) values($1,$2) returning id`, "GBP", "USD").Scan(&pairID)
	require.NoError(t, err)
	updID := uuid.New()
	// Use a value with more decimals to verify rounding happens in query (4 decimals)
	_, err = pool.Exec(ctx, `insert into fx_rate_updates(pair_id, update_id, status, value) values ($1,$2,'applied',$3)`, pairID, updID, 0.999949)
	require.NoError(t, err)

	rate, status, err := repo.GetByUpdateID(ctx, updID)
	require.NoError(t, err)
	require.Equal(t, pairID, rate.PairID)
	require.Equal(t, domain.StatusApplied, status)
	require.InDelta(t, 0.9999, rate.Value, 0.00001)
}

func TestRateRepository_GetByUpdateID_DBError(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateRepository(pool)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := repo.GetByUpdateID(ctx, uuid.New())
	require.Error(t, err)
	require.NotErrorIs(t, err, domain.ErrRateNotFound)
}

// ---------- RateUpdateRepository tests ----------

func TestRateUpdateRepository_ScheduleNewOrGetExisting_NewAndExisting(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateUpdateRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `insert into currencies(code) values ('USD'),('EUR')`)
	require.NoError(t, err)

	// First schedule — creates pair and pending update.
	upd1, err := repo.ScheduleNewOrGetExisting(ctx, "USD", "EUR")
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, upd1)

	// Verify one pending update exists and matches returned ID.
	var got uuid.UUID
	err = pool.QueryRow(ctx, `select update_id from fx_rate_updates where status='pending'`).Scan(&got)
	require.NoError(t, err)
	require.Equal(t, upd1, got)

	// Second schedule — should return the same pending update id (idempotent while pending).
	upd2, err := repo.ScheduleNewOrGetExisting(ctx, "USD", "EUR")
	require.NoError(t, err)
	require.Equal(t, upd1, upd2)
}

func TestRateUpdateRepository_ScheduleNewOrGetExisting_InvalidCurrency_Error(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateUpdateRepository(pool)
	ctx := context.Background()

	// FOO/BAR are intentionally not seeded; expect FK violation → wrapped error.
	_, err := repo.ScheduleNewOrGetExisting(ctx, "FOO", "BAR")
	require.Error(t, err)
}

func TestRateUpdateRepository_ScheduleNewOrGetExisting_DBError(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateUpdateRepository(pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := pool.Exec(context.Background(), `insert into currencies(code) values ('USD'),('JPY')`)
	require.NoError(t, err)

	_, err = repo.ScheduleNewOrGetExisting(ctx, "USD", "JPY")
	require.Error(t, err)
}

func TestRateUpdateRepository_GetPending_Empty(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateUpdateRepository(pool)
	ctx := context.Background()

	pending, err := repo.GetPending(ctx)
	require.NoError(t, err)
	require.Empty(t, pending)
}

func TestRateUpdateRepository_GetPending_OnlyPendingReturned(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateUpdateRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `insert into currencies(code) values ('USD'),('MXN'),('EUR'),('GBP')`)
	require.NoError(t, err)

	// Insert two pairs and a mix of statuses.
	var p1, p2 int64
	require.NoError(t, pool.QueryRow(ctx, `insert into fx_pairs(base, quote) values('USD','MXN') returning id`).Scan(&p1))
	require.NoError(t, pool.QueryRow(ctx, `insert into fx_pairs(base, quote) values('EUR','GBP') returning id`).Scan(&p2))

	u1 := uuid.New()
	u2 := uuid.New()
	_, err = pool.Exec(ctx, `insert into fx_rate_updates(pair_id, update_id, status) values ($1,$2,'pending')`, p1, u1)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `insert into fx_rate_updates(pair_id, update_id, status, value) values ($1,$2,'applied', 3.14)`, p2, u2)
	require.NoError(t, err)

	pending, err := repo.GetPending(ctx)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, u1, pending[0].UpdateID)
	require.Equal(t, p1, pending[0].PairID)
	require.Equal(t, "USD", pending[0].Base)
	require.Equal(t, "MXN", pending[0].Quote)
}

func TestRateUpdateRepository_GetPending_DBError(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateUpdateRepository(pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := repo.GetPending(ctx)
	require.Error(t, err)
}

func TestRateUpdateRepository_ApplyUpdates_EmptyNoop(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateUpdateRepository(pool)
	ctx := context.Background()

	err1 := repo.ApplyUpdates(ctx, nil)
	require.NoError(t, err1)
	err2 := repo.ApplyUpdates(ctx, make([]domain.AppliedRateUpdate, 0))
	require.NoError(t, err2)
}

func TestRateUpdateRepository_ApplyUpdates_ApplyAndPropagateToLastRates(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateUpdateRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `insert into currencies(code) values ('JPY'),('USD')`)
	require.NoError(t, err)

	// Arrange: one pending update.
	var pairID int64
	require.NoError(t, pool.QueryRow(ctx, `insert into fx_pairs(base, quote) values('JPY','USD') returning id`).Scan(&pairID))
	upd := uuid.New()
	_, err = pool.Exec(ctx, `insert into fx_rate_updates(pair_id, update_id, status) values ($1,$2,'pending')`, pairID, upd)
	require.NoError(t, err)

	// Apply update.
	err = repo.ApplyUpdates(ctx, []domain.AppliedRateUpdate{{UpdateID: upd, PairID: pairID, Value: 123.4567}})
	require.NoError(t, err)

	// Verify fx_rate_updates changed to applied with value.
	var status domain.RateUpdateStatus
	var value float64
	err = pool.QueryRow(ctx, `select status, value from fx_rate_updates where update_id = $1`, upd).Scan(&status, &value)
	require.NoError(t, err)
	require.Equal(t, domain.StatusApplied, status)
	require.InDelta(t, 123.4567, value, 0.0000001)

	// Verify fx_last_rates upserted.
	var lr float64
	err = pool.QueryRow(ctx, `select value from fx_last_rates where pair_id = $1`, pairID).Scan(&lr)
	require.NoError(t, err)
	require.InDelta(t, 123.4567, lr, 0.0000001)
}

func TestRateUpdateRepository_ApplyUpdates_PartialApply(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateUpdateRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `insert into currencies(code) values ('EUR'),('JPY'),('GBP')`)
	require.NoError(t, err)

	var p1, p2 int64
	require.NoError(t, pool.QueryRow(ctx, `insert into fx_pairs(base, quote) values('EUR','JPY') returning id`).Scan(&p1))
	require.NoError(t, pool.QueryRow(ctx, `insert into fx_pairs(base, quote) values('GBP','EUR') returning id`).Scan(&p2))
	u1 := uuid.New()
	u2 := uuid.New()
	_, err = pool.Exec(ctx, `insert into fx_rate_updates(pair_id, update_id, status) values ($1,$2,'pending')`, p1, u1)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `insert into fx_rate_updates(pair_id, update_id, status) values ($1,$2,'pending')`, p2, u2)
	require.NoError(t, err)

	// Apply only one of them.
	err = repo.ApplyUpdates(ctx, []domain.AppliedRateUpdate{{UpdateID: u1, PairID: p1, Value: 1.5}})
	require.NoError(t, err)

	// u1 should be applied, u2 should remain pending.
	var s1, s2 domain.RateUpdateStatus
	require.NoError(t, pool.QueryRow(ctx, `select status from fx_rate_updates where update_id = $1`, u1).Scan(&s1))
	require.NoError(t, pool.QueryRow(ctx, `select status from fx_rate_updates where update_id = $1`, u2).Scan(&s2))
	require.Equal(t, domain.StatusApplied, s1)
	require.Equal(t, domain.StatusPending, s2)
}

func TestRateUpdateRepository_ApplyUpdates_JSONMarshalError_NaN(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateUpdateRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `insert into currencies(code) values ('USD'),('GBP')`)
	require.NoError(t, err)

	// Prepare a pending update so that only JSON marshal fails (due to NaN) before touching DB.
	var pairID int64
	require.NoError(t, pool.QueryRow(ctx, `insert into fx_pairs(base, quote) values('USD','GBP') returning id`).Scan(&pairID))
	upd := uuid.New()
	_, err = pool.Exec(ctx, `insert into fx_rate_updates(pair_id, update_id, status) values ($1,$2,'pending')`, pairID, upd)
	require.NoError(t, err)

	err = repo.ApplyUpdates(ctx, []domain.AppliedRateUpdate{{UpdateID: upd, PairID: pairID, Value: math.NaN()}})
	require.Error(t, err)
}

func TestRateUpdateRepository_ApplyUpdates_DBError_BeginTx(t *testing.T) {
	pool := setupPostgres(t)
	repo := postgres.NewRateUpdateRepository(pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := repo.ApplyUpdates(ctx, []domain.AppliedRateUpdate{{UpdateID: uuid.New(), PairID: 1, Value: 1.0}})
	require.Error(t, err)
}
