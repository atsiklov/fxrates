package rate

import (
	"context"
	"errors"
	"testing"
	"time"

	"fxrates/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Testify mocks ---

type MockRateUpdateRepository struct{ mock.Mock }

func (m *MockRateUpdateRepository) ScheduleNewOrGetExisting(ctx context.Context, base string, quote string) (uuid.UUID, error) {
	args := m.Called(ctx, base, quote)
	id, _ := args.Get(0).(uuid.UUID)
	return id, args.Error(1)
}

func (m *MockRateUpdateRepository) GetPending(ctx context.Context) ([]domain.PendingRateUpdate, error) {
	args := m.Called(ctx)
	updates, _ := args.Get(0).([]domain.PendingRateUpdate)
	return updates, args.Error(1)
}

func (m *MockRateUpdateRepository) ApplyUpdates(ctx context.Context, rates []domain.AppliedRateUpdate) error {
	args := m.Called(ctx, rates)
	return args.Error(0)
}

type MockRateRepository struct{ mock.Mock }

func (m *MockRateRepository) GetByCodes(ctx context.Context, base string, quote string) (domain.Rate, error) {
	args := m.Called(ctx, base, quote)
	r, _ := args.Get(0).(domain.Rate)
	return r, args.Error(1)
}

func (m *MockRateRepository) GetByUpdateID(ctx context.Context, updateID uuid.UUID) (domain.Rate, domain.RateUpdateStatus, error) {
	args := m.Called(ctx, updateID)
	r, _ := args.Get(0).(domain.Rate)
	status, _ := args.Get(1).(domain.RateUpdateStatus)
	return r, status, args.Error(2)
}

type MockRateUpdateCache struct{ mock.Mock }

func (m *MockRateUpdateCache) Get(pair domain.RatePair) (uuid.UUID, bool) {
	args := m.Called(pair)
	id, _ := args.Get(0).(uuid.UUID)
	return id, args.Bool(1)
}

func (m *MockRateUpdateCache) Set(pair domain.RatePair, updateID uuid.UUID) {
	m.Called(pair, updateID)
}

func (m *MockRateUpdateCache) CleanBatch(pairs []domain.RatePair) {
	m.Called(pairs)
}

// --- ScheduleUpdate ---

func TestService_ScheduleUpdate_Success(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockRateRepo := new(MockRateRepository)
	mockCache := new(MockRateUpdateCache)
	svc := NewService(mockUpdatesRepo, mockRateRepo, mockCache)

	ctx := context.Background()
	updateID := uuid.New()
	pair := domain.RatePair{Base: "USD", Quote: "EUR"}

	mockCache.On("Get", pair).Return(uuid.Nil, false).Once()
	mockUpdatesRepo.On("ScheduleNewOrGetExisting", mock.Anything, "USD", "EUR").Return(updateID, nil).Once()
	mockCache.On("Set", pair, updateID).Return().Once()

	id, err := svc.ScheduleUpdate(ctx, "USD", "EUR")

	require.NoError(t, err)
	require.Equal(t, updateID, id)
	mockUpdatesRepo.AssertExpectations(t)
	mockRateRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestService_ScheduleUpdate_Error(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockRateRepo := new(MockRateRepository)
	mockCache := new(MockRateUpdateCache)
	svc := NewService(mockUpdatesRepo, mockRateRepo, mockCache)

	ctx := context.Background()
	wantErr := errors.New("db temporarily unavailable")
	pair := domain.RatePair{Base: "USD", Quote: "EUR"}

	mockCache.On("Get", pair).Return(uuid.Nil, false).Once()
	mockUpdatesRepo.On("ScheduleNewOrGetExisting", mock.Anything, "USD", "EUR").Return(uuid.Nil, wantErr).Once()

	id, err := svc.ScheduleUpdate(ctx, "USD", "EUR")

	require.Error(t, err)
	require.Equal(t, uuid.Nil, id)
	require.Equal(t, wantErr, err)
	mockUpdatesRepo.AssertExpectations(t)
	mockRateRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestService_ScheduleUpdate_UsesCacheHit(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockRateRepo := new(MockRateRepository)
	mockCache := new(MockRateUpdateCache)
	svc := NewService(mockUpdatesRepo, mockRateRepo, mockCache)

	ctx := context.Background()
	updateID := uuid.New()
	pair := domain.RatePair{Base: "USD", Quote: "EUR"}

	mockCache.On("Get", pair).Return(updateID, true).Once()

	id, err := svc.ScheduleUpdate(ctx, "USD", "EUR")

	require.NoError(t, err)
	require.Equal(t, updateID, id)
	mockUpdatesRepo.AssertNotCalled(t, "ScheduleNewOrGetExisting", mock.Anything, mock.Anything, mock.Anything)
	mockCache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything)
}

// --- GetByUpdateID ---

func TestService_GetByUpdateID_StatusApplied(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockRateRepo := new(MockRateRepository)
	svc := NewService(mockUpdatesRepo, mockRateRepo, nil)

	ctx := context.Background()
	updateID := uuid.New()
	fixedTime := time.Date(2024, 11, 15, 10, 9, 8, 0, time.UTC)
	rate := domain.Rate{Base: "USD", Quote: "EUR", Value: 1.2345, UpdatedAt: fixedTime}

	mockRateRepo.On("GetByUpdateID", mock.Anything, updateID).Return(rate, domain.StatusApplied, nil).Once()

	view, err := svc.GetByUpdateID(ctx, updateID)

	require.NoError(t, err)
	require.Equal(t, "USD", view.Base)
	require.Equal(t, "EUR", view.Quote)
	require.Equal(t, domain.StatusApplied, view.Status)
	require.NotNil(t, view.Value)
	require.InDelta(t, 1.2345, *view.Value, 1e-9)
	require.NotNil(t, view.UpdatedAt)
	require.True(t, view.UpdatedAt.Equal(fixedTime))
	mockRateRepo.AssertExpectations(t)
	mockUpdatesRepo.AssertExpectations(t)
}

func TestService_GetByUpdateID_StatusPending(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockRateRepo := new(MockRateRepository)
	svc := NewService(mockUpdatesRepo, mockRateRepo, nil)

	ctx := context.Background()
	updateID := uuid.New()
	rate := domain.Rate{Base: "GBP", Quote: "JPY"}

	mockRateRepo.On("GetByUpdateID", mock.Anything, updateID).Return(rate, domain.StatusPending, nil).Once()

	view, err := svc.GetByUpdateID(ctx, updateID)

	require.NoError(t, err)
	require.Equal(t, "GBP", view.Base)
	require.Equal(t, "JPY", view.Quote)
	require.Equal(t, domain.StatusPending, view.Status)
	require.Nil(t, view.Value)
	require.Nil(t, view.UpdatedAt)
	mockRateRepo.AssertExpectations(t)
	mockUpdatesRepo.AssertExpectations(t)
}

func TestService_GetByUpdateID_UnknownStatus(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockRateRepo := new(MockRateRepository)
	svc := NewService(mockUpdatesRepo, mockRateRepo, nil)

	ctx := context.Background()
	updateID := uuid.New()
	rate := domain.Rate{Base: "AUD", Quote: "CAD"}

	unknown := domain.RateUpdateStatus("unknown")
	mockRateRepo.On("GetByUpdateID", mock.Anything, updateID).Return(rate, unknown, nil).Once()

	_, err := svc.GetByUpdateID(ctx, updateID)

	require.Error(t, err)
	require.EqualError(t, err, "unknown rate update status: \"unknown\"")
	mockRateRepo.AssertExpectations(t)
	mockUpdatesRepo.AssertExpectations(t)
}

func TestService_GetByUpdateID_RepoError(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockRateRepo := new(MockRateRepository)
	svc := NewService(mockUpdatesRepo, mockRateRepo, nil)

	ctx := context.Background()
	updateID := uuid.New()
	wantErr := errors.New("db query failed")

	mockRateRepo.On("GetByUpdateID", mock.Anything, updateID).Return(domain.Rate{}, "", wantErr).Once()

	_, err := svc.GetByUpdateID(ctx, updateID)

	require.Error(t, err)
	require.Equal(t, wantErr, err)
	mockRateRepo.AssertExpectations(t)
	mockUpdatesRepo.AssertExpectations(t)
}

// --- GetByCodes ---

func TestService_GetByCodes_Success(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockRateRepo := new(MockRateRepository)
	svc := NewService(mockUpdatesRepo, mockRateRepo, nil)

	ctx := context.Background()
	fixedTime := time.Date(2024, 10, 1, 12, 0, 0, 0, time.UTC)
	rate := domain.Rate{Base: "USD", Quote: "CHF", Value: 0.915, UpdatedAt: fixedTime}

	mockRateRepo.On("GetByCodes", mock.Anything, "USD", "CHF").Return(rate, nil).Once()

	view, err := svc.GetByCodes(ctx, "USD", "CHF")

	require.NoError(t, err)
	require.Equal(t, "USD", view.Base)
	require.Equal(t, "CHF", view.Quote)
	require.NotNil(t, view.Value)
	require.InDelta(t, 0.915, *view.Value, 1e-9)
	require.NotNil(t, view.UpdatedAt)
	require.True(t, view.UpdatedAt.Equal(fixedTime))
	mockRateRepo.AssertExpectations(t)
	mockUpdatesRepo.AssertExpectations(t)
}

func TestService_GetByCodes_Error(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockRateRepo := new(MockRateRepository)
	svc := NewService(mockUpdatesRepo, mockRateRepo, nil)

	ctx := context.Background()
	wantErr := domain.ErrRateNotFound

	mockRateRepo.On("GetByCodes", mock.Anything, "USD", "PLN").Return(domain.Rate{}, wantErr).Once()

	_, err := svc.GetByCodes(ctx, "USD", "PLN")

	require.Error(t, err)
	require.Equal(t, wantErr, err)
	mockRateRepo.AssertExpectations(t)
	mockUpdatesRepo.AssertExpectations(t)
}
