package rate

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"

	"fxrates/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockRateClient struct{ mock.Mock }

func (m *MockRateClient) GetExchangeRates(ctx context.Context, code string) (map[string]float64, error) {
	args := m.Called(ctx, code)
	rates, _ := args.Get(0).(map[string]float64)
	return rates, args.Error(1)
}

// --- getUniquePairs ---

func TestGetUniquePairs_SkipsReversedAndSetsDefaults(t *testing.T) {
	pending := []domain.PendingRateUpdate{
		{UpdateID: uuid.New(), PairID: 1, Base: "USD", Quote: "EUR"},
		{UpdateID: uuid.New(), PairID: 2, Base: "USD", Quote: "MXN"},
		{UpdateID: uuid.New(), PairID: 3, Base: "MXN", Quote: "EUR"},
		{UpdateID: uuid.New(), PairID: 4, Base: "EUR", Quote: "USD"}, // reversed, should be skipped
	}

	pairs := getUniquePairs(pending)

	require.Len(t, pairs, 3)
	_, hasUSDEUR := pairs[pair{Base: "USD", Quote: "EUR"}]
	_, hasUSDMXN := pairs[pair{Base: "USD", Quote: "MXN"}]
	_, hasMXNEUR := pairs[pair{Base: "MXN", Quote: "EUR"}]
	_, hasEURUSD := pairs[pair{Base: "EUR", Quote: "USD"}]

	require.True(t, hasUSDEUR)
	require.True(t, hasUSDMXN)
	require.True(t, hasMXNEUR)
	require.False(t, hasEURUSD)

	// values are initialized to -1
	require.Equal(t, float64(-1), pairs[pair{Base: "USD", Quote: "EUR"}])
	require.Equal(t, float64(-1), pairs[pair{Base: "USD", Quote: "MXN"}])
	require.Equal(t, float64(-1), pairs[pair{Base: "MXN", Quote: "EUR"}])
}

// --- getUniqueBases ---

func TestGetUniqueBases_CollectsUnique(t *testing.T) {
	pairs := map[pair]float64{
		{Base: "USD", Quote: "EUR"}: -1,
		{Base: "USD", Quote: "PLN"}: -1,
		{Base: "EUR", Quote: "GBP"}: -1,
	}

	bases := getUniqueBases(pairs)
	sort.Strings(bases)
	require.Equal(t, []string{"EUR", "USD"}, bases)
}

// --- processBase ---

func TestProcessBase_ErrorFromClient_DoesNotModify(t *testing.T) {
	mockClient := new(MockRateClient)
	pairs := map[pair]float64{
		{Base: "USD", Quote: "EUR"}: -1,
	}
	mu := new(sync.Mutex)

	mockClient.On("GetExchangeRates", mock.Anything, "USD").Return(nil, errors.New("timeout")).Once()

	processBase(context.Background(), 1, "USD", mockClient, pairs, mu)

	require.Equal(t, float64(-1), pairs[pair{Base: "USD", Quote: "EUR"}])
	mockClient.AssertExpectations(t)
}

func TestProcessBase_UpdatesMatchingPairsOnly(t *testing.T) {
	mockClient := new(MockRateClient)
	pairs := map[pair]float64{
		{Base: "USD", Quote: "EUR"}: -1,
		{Base: "USD", Quote: "PLN"}: -1,
		{Base: "EUR", Quote: "JPY"}: -1,
	}
	mu := new(sync.Mutex)

	mockClient.On("GetExchangeRates", mock.Anything, "USD").Return(map[string]float64{
		"EUR": 1.2,
		"PLN": 4.0,
		"JPY": 150,
	}, nil).Once()

	processBase(context.Background(), 2, "USD", mockClient, pairs, mu)

	require.InDelta(t, 1.2, pairs[pair{Base: "USD", Quote: "EUR"}], 1e-9)
	require.InDelta(t, 4.0, pairs[pair{Base: "USD", Quote: "PLN"}], 1e-9)
	// pair with base EUR should not be modified by processing base USD
	require.Equal(t, float64(-1), pairs[pair{Base: "EUR", Quote: "JPY"}])
	mockClient.AssertExpectations(t)
}

// --- runWorker ---

func TestRunWorker_ProcessesQueue(t *testing.T) {
	mockClient := new(MockRateClient)
	queue := make(chan string, 2)
	queue <- "USD"
	queue <- "EUR"
	close(queue)

	pairs := map[pair]float64{
		{Base: "USD", Quote: "EUR"}: -1,
		{Base: "EUR", Quote: "USD"}: -1,
	}
	mu := new(sync.Mutex)

	mockClient.On("GetExchangeRates", mock.Anything, "USD").Return(map[string]float64{"EUR": 1.3}, nil).Once()
	mockClient.On("GetExchangeRates", mock.Anything, "EUR").Return(map[string]float64{"USD": 0.77}, nil).Once()

	done := make(chan struct{})
	go func() {
		runWorker(context.Background(), 7, queue, mockClient, pairs, mu)
		close(done)
	}()

	<-done

	require.InDelta(t, 1.3, pairs[pair{Base: "USD", Quote: "EUR"}], 1e-9)
	require.InDelta(t, 0.77, pairs[pair{Base: "EUR", Quote: "USD"}], 1e-9)
	mockClient.AssertExpectations(t)
}

// --- processInParallel ---

func TestProcessInParallel_UpdatesAllBases(t *testing.T) {
	mockClient := new(MockRateClient)
	pairs := map[pair]float64{
		{Base: "USD", Quote: "EUR"}: -1,
		{Base: "USD", Quote: "PLN"}: -1,
		{Base: "EUR", Quote: "GBP"}: -1,
	}

	mockClient.On("GetExchangeRates", mock.Anything, "USD").Return(map[string]float64{"EUR": 1.11, "PLN": 3.99}, nil).Once()
	mockClient.On("GetExchangeRates", mock.Anything, "EUR").Return(map[string]float64{"GBP": 0.86}, nil).Once()

	processInParallel(context.Background(), mockClient, pairs)

	require.InDelta(t, 1.11, pairs[pair{Base: "USD", Quote: "EUR"}], 1e-9)
	require.InDelta(t, 3.99, pairs[pair{Base: "USD", Quote: "PLN"}], 1e-9)
	require.InDelta(t, 0.86, pairs[pair{Base: "EUR", Quote: "GBP"}], 1e-9)
	mockClient.AssertExpectations(t)
}

// --- doUpdateRates ---

func TestDoUpdateRates_AppliesDirectAndReversedAndSkipsMissing(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	pending := []domain.PendingRateUpdate{
		{UpdateID: uuid.New(), PairID: 1, Base: "USD", Quote: "EUR"}, // direct present
		{UpdateID: uuid.New(), PairID: 2, Base: "EUR", Quote: "PLN"}, // reversed present via PLN/EUR
		{UpdateID: uuid.New(), PairID: 3, Base: "GBP", Quote: "JPY"}, // missing -> skip
		{UpdateID: uuid.New(), PairID: 4, Base: "AUD", Quote: "NZD"}, // non-positive -> skip
	}
	pairs := map[pair]float64{
		{Base: "USD", Quote: "EUR"}: 0.9,  // direct
		{Base: "PLN", Quote: "EUR"}: 4.0,  // reversed available => 1/4.0
		{Base: "AUD", Quote: "NZD"}: -1.0, // non-positive
	}

	mockUpdatesRepo.
		On("ApplyUpdates", mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			applied, ok := args.Get(1).([]domain.AppliedRateUpdate) // чекаем аргументы мока
			require.True(t, ok)
			require.Len(t, applied, 2)

			require.InDelta(t, 0.9, applied[0].Value, 1e-9)     // для пары USD/EUR
			require.InDelta(t, 1.0/4.0, applied[1].Value, 1e-9) // для пары EUR/PLN, которую ревёрснули и нашли в pairs
		}).Once()

	count, err := doUpdateRates(context.Background(), pending, pairs, mockUpdatesRepo)

	require.NoError(t, err)
	require.Equal(t, 2, count)
	mockUpdatesRepo.AssertExpectations(t)
}

func TestDoUpdateRates_NoApplicableUpdates_DoesNotCallRepo(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	pending := []domain.PendingRateUpdate{
		{UpdateID: uuid.New(), PairID: 10, Base: "USD", Quote: "EUR"},
	}
	pairs := map[pair]float64{
		{Base: "USD", Quote: "EUR"}: -1, // not usable
	}

	count, err := doUpdateRates(context.Background(), pending, pairs, mockUpdatesRepo)

	require.NoError(t, err)
	require.Equal(t, 0, count)
	mockUpdatesRepo.AssertNotCalled(t, "ApplyUpdates", mock.Anything, mock.Anything)
}

func TestDoUpdateRates_ApplyUpdatesError_Propagates(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	pending := []domain.PendingRateUpdate{
		{UpdateID: uuid.New(), PairID: 5, Base: "USD", Quote: "EUR"},
	}
	pairs := map[pair]float64{
		{Base: "USD", Quote: "EUR"}: 1.01,
	}
	wantErr := errors.New("db fail")

	mockUpdatesRepo.On("ApplyUpdates", mock.Anything, mock.Anything).Return(wantErr).Once()

	count, err := doUpdateRates(context.Background(), pending, pairs, mockUpdatesRepo)

	require.Error(t, err)
	require.ErrorContains(t, err, "failed to update rates")
	require.Equal(t, 0, count)
	mockUpdatesRepo.AssertExpectations(t)
}

// --- UpdatePendingRates ---

func TestUpdatePendingRates_GetPendingError(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockClient := new(MockRateClient)
	wantErr := errors.New("db unavailable")

	mockUpdatesRepo.On("GetPending", mock.Anything).Return(nil, wantErr).Once()

	err := UpdatePendingRates(context.Background(), "exec-1", mockUpdatesRepo, mockClient)

	require.Error(t, err)
	require.ErrorContains(t, err, "failed to get pending rates")
	mockUpdatesRepo.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestUpdatePendingRates_NoPending(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockClient := new(MockRateClient)

	mockUpdatesRepo.On("GetPending", mock.Anything).Return([]domain.PendingRateUpdate{}, nil).Once()

	err := UpdatePendingRates(context.Background(), "exec-2", mockUpdatesRepo, mockClient)

	require.NoError(t, err)
	mockUpdatesRepo.AssertExpectations(t)
	mockUpdatesRepo.AssertNotCalled(t, "ApplyUpdates", mock.Anything, mock.Anything)
	mockClient.AssertExpectations(t)
}

func TestUpdatePendingRates_Success_AppliesValues(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockClient := new(MockRateClient)

	p1 := domain.PendingRateUpdate{UpdateID: uuid.New(), PairID: 1, Base: "USD", Quote: "EUR"}
	p2 := domain.PendingRateUpdate{UpdateID: uuid.New(), PairID: 2, Base: "EUR", Quote: "PLN"}
	mockUpdatesRepo.On("GetPending", mock.Anything).Return([]domain.PendingRateUpdate{p1, p2}, nil).Once()

	mockClient.On("GetExchangeRates", mock.Anything, "USD").Return(map[string]float64{"EUR": 1.23}, nil).Once()
	mockClient.On("GetExchangeRates", mock.Anything, "EUR").Return(map[string]float64{"PLN": 4.56}, nil).Once()

	mockUpdatesRepo.On("ApplyUpdates", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		updates := args.Get(1).([]domain.AppliedRateUpdate)
		require.Len(t, updates, 2)
		// Sort by PairID for deterministic check
		if updates[0].PairID > updates[1].PairID {
			updates[0], updates[1] = updates[1], updates[0]
		}
		require.Equal(t, int64(1), updates[0].PairID)
		require.InDelta(t, 1.23, updates[0].Value, 1e-9)
		require.Equal(t, int64(2), updates[1].PairID)
		require.InDelta(t, 4.56, updates[1].Value, 1e-9)
	}).Once()

	err := UpdatePendingRates(context.Background(), "exec-3", mockUpdatesRepo, mockClient)

	require.NoError(t, err)
	mockUpdatesRepo.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

func TestUpdatePendingRates_ApplyUpdatesError_Propagates(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockClient := new(MockRateClient)

	p1 := domain.PendingRateUpdate{UpdateID: uuid.New(), PairID: 1, Base: "USD", Quote: "EUR"}
	mockUpdatesRepo.On("GetPending", mock.Anything).Return([]domain.PendingRateUpdate{p1}, nil).Once()

	mockClient.On("GetExchangeRates", mock.Anything, "USD").Return(map[string]float64{"EUR": 1.11}, nil).Once()

	wantErr := errors.New("apply failed")
	mockUpdatesRepo.On("ApplyUpdates", mock.Anything, mock.Anything).Return(wantErr).Once()

	err := UpdatePendingRates(context.Background(), "exec-4", mockUpdatesRepo, mockClient)

	require.Error(t, err)
	require.ErrorContains(t, err, "failed to update rates")
	mockUpdatesRepo.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}
