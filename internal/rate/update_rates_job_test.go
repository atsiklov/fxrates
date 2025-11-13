package rate

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sort"
	"testing"

	"fxrates/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
	_, hasUSDEUR := pairs[domain.RatePair{Base: "USD", Quote: "EUR"}]
	_, hasUSDMXN := pairs[domain.RatePair{Base: "USD", Quote: "MXN"}]
	_, hasMXNEUR := pairs[domain.RatePair{Base: "MXN", Quote: "EUR"}]
	_, hasEURUSD := pairs[domain.RatePair{Base: "EUR", Quote: "USD"}]

	require.True(t, hasUSDEUR)
	require.True(t, hasUSDMXN)
	require.True(t, hasMXNEUR)
	require.False(t, hasEURUSD)

	// values are initialized to -1
	require.Equal(t, struct{}{}, pairs[domain.RatePair{Base: "USD", Quote: "EUR"}])
	require.Equal(t, struct{}{}, pairs[domain.RatePair{Base: "USD", Quote: "MXN"}])
	require.Equal(t, struct{}{}, pairs[domain.RatePair{Base: "MXN", Quote: "EUR"}])
}

// --- getUniqueBases ---

func TestGetUniqueBases_CollectsUnique(t *testing.T) {
	pairs := map[domain.RatePair]struct{}{
		{Base: "USD", Quote: "EUR"}: {},
		{Base: "USD", Quote: "PLN"}: {},
		{Base: "EUR", Quote: "GBP"}: {},
	}

	bases := slices.Collect(maps.Keys(getUniqueBases(pairs)))
	sort.Strings(bases)
	require.Len(t, bases, 2)
	require.Equal(t, []string{"EUR", "USD"}, bases)
}

// --- processBase ---

func TestProcessBase_ErrorFromClient_DoesNotModify(t *testing.T) {
	mockClient := new(MockRateClient)
	pairs := map[domain.RatePair]struct{}{
		{Base: "USD", Quote: "EUR"}: {},
	}
	mockClient.On("GetExchangeRates", mock.Anything, "USD").Return(nil, errors.New("timeout")).Once()

	updates := make(chan rateUpdate, 1)
	processBase(context.Background(), 1, "USD", mockClient, pairs, updates)

	select {
	case <-updates:
		t.Fatal("expected no updates to be emitted")
	default:
	}
	mockClient.AssertExpectations(t)
}

func TestProcessBase_UpdatesMatchingPairsOnly(t *testing.T) {
	mockClient := new(MockRateClient)
	pairs := map[domain.RatePair]struct{}{
		{Base: "USD", Quote: "EUR"}: {},
		{Base: "USD", Quote: "PLN"}: {},
		{Base: "EUR", Quote: "JPY"}: {},
	}
	mockClient.On("GetExchangeRates", mock.Anything, "USD").Return(map[string]float64{
		"EUR": 1.2,
		"PLN": 4.0,
		"JPY": 150,
	}, nil).Once()

	updates := make(chan rateUpdate, len(pairs))

	processBase(context.Background(), 2, "USD", mockClient, pairs, updates)
	close(updates)

	results := map[domain.RatePair]float64{}
	for upd := range updates {
		results[upd.Pair] = upd.Value
	}
	require.InDelta(t, 1.2, results[domain.RatePair{Base: "USD", Quote: "EUR"}], 1e-9)
	require.InDelta(t, 4.0, results[domain.RatePair{Base: "USD", Quote: "PLN"}], 1e-9)
	require.NotContains(t, results, domain.RatePair{Base: "EUR", Quote: "JPY"})
	mockClient.AssertExpectations(t)
}

// --- runWorker ---

func TestRunWorker_ProcessesQueue(t *testing.T) {
	mockClient := new(MockRateClient)
	queue := make(chan string, 2)
	queue <- "USD"
	queue <- "EUR"
	close(queue)

	pairs := map[domain.RatePair]struct{}{
		{Base: "USD", Quote: "EUR"}: {},
		{Base: "EUR", Quote: "USD"}: {},
	}

	mockClient.On("GetExchangeRates", mock.Anything, "USD").Return(map[string]float64{"EUR": 1.3}, nil).Once()
	mockClient.On("GetExchangeRates", mock.Anything, "EUR").Return(map[string]float64{"USD": 0.77}, nil).Once()

	done := make(chan struct{})
	updates := make(chan rateUpdate, 4)
	go func() {
		runWorker(context.Background(), 7, queue, mockClient, pairs, updates)
		close(done)
	}()

	<-done
	close(updates)

	results := make(map[domain.RatePair]float64)
	for upd := range updates {
		results[upd.Pair] = upd.Value
	}
	require.InDelta(t, 1.3, results[domain.RatePair{Base: "USD", Quote: "EUR"}], 1e-9)
	require.InDelta(t, 0.77, results[domain.RatePair{Base: "EUR", Quote: "USD"}], 1e-9)
	mockClient.AssertExpectations(t)
}

// --- processInParallel ---

func TestProcessInParallel_UpdatesAllBasesAndReturnsPairValueMap(t *testing.T) {
	mockClient := new(MockRateClient)
	pairs := map[domain.RatePair]struct{}{
		{Base: "USD", Quote: "EUR"}: {},
		{Base: "USD", Quote: "PLN"}: {},
		{Base: "EUR", Quote: "GBP"}: {},
	}

	mockClient.On("GetExchangeRates", mock.Anything, "USD").Return(map[string]float64{"EUR": 1.11, "PLN": 3.99}, nil).Once()
	mockClient.On("GetExchangeRates", mock.Anything, "EUR").Return(map[string]float64{"GBP": 0.86}, nil).Once()

	pairValueMap := processInParallel(context.Background(), mockClient, pairs)

	require.InDelta(t, 1.11, pairValueMap[domain.RatePair{Base: "USD", Quote: "EUR"}], 1e-9)
	require.InDelta(t, 3.99, pairValueMap[domain.RatePair{Base: "USD", Quote: "PLN"}], 1e-9)
	require.InDelta(t, 0.86, pairValueMap[domain.RatePair{Base: "EUR", Quote: "GBP"}], 1e-9)
	mockClient.AssertExpectations(t)
}

// --- doUpdateRates ---

func TestDoUpdateRates_AppliesDirectAndReversedAndSkipsMissing(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	cacheMock := new(MockRateUpdateCache)
	pending := []domain.PendingRateUpdate{
		{UpdateID: uuid.New(), PairID: 1, Base: "USD", Quote: "EUR"}, // direct present
		{UpdateID: uuid.New(), PairID: 2, Base: "EUR", Quote: "PLN"}, // reversed present via PLN/EUR
		{UpdateID: uuid.New(), PairID: 3, Base: "GBP", Quote: "JPY"}, // missing -> skip
		{UpdateID: uuid.New(), PairID: 4, Base: "AUD", Quote: "NZD"}, // non-positive -> skip
	}
	pairValueMap := map[domain.RatePair]float64{
		{Base: "USD", Quote: "EUR"}: 0.9, // direct
		{Base: "PLN", Quote: "EUR"}: 4.0, // reversed available => 1/4.0
	}

	mockUpdatesRepo.
		On("ApplyUpdates", mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			applied, ok := args.Get(1).([]domain.AppliedRateUpdate)
			require.True(t, ok)
			require.Len(t, applied, 2)

			require.InDelta(t, 0.9, applied[0].Value, 1e-9)
			require.InDelta(t, 1.0/4.0, applied[1].Value, 1e-9)
		}).Once()

	expectedPairs := []domain.RatePair{
		{Base: "USD", Quote: "EUR"},
		{Base: "EUR", Quote: "PLN"},
	}
	cacheMock.On("CleanBatch", mock.MatchedBy(func(pairs []domain.RatePair) bool {
		return assert.ElementsMatch(t, expectedPairs, pairs)
	})).Return().Once()

	count, err := doUpdateRates(context.Background(), pending, pairValueMap, mockUpdatesRepo, cacheMock)

	require.NoError(t, err)
	require.Equal(t, 2, count)
	mockUpdatesRepo.AssertExpectations(t)
	cacheMock.AssertExpectations(t)
}

func TestDoUpdateRates_NoApplicableUpdates_DoesNotCallRepoAndDoesNotCleanCache(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	cacheMock := new(MockRateUpdateCache)
	pending := []domain.PendingRateUpdate{
		{UpdateID: uuid.New(), PairID: 10, Base: "USD", Quote: "FOO"},
	}
	pairValueMap := map[domain.RatePair]float64{
		{Base: "USD", Quote: "EUR"}: 1.47,
	}

	count, err := doUpdateRates(context.Background(), pending, pairValueMap, mockUpdatesRepo, cacheMock)

	require.NoError(t, err)
	require.Equal(t, 0, count)
	mockUpdatesRepo.AssertNotCalled(t, "ApplyUpdates", mock.Anything, mock.Anything)
	cacheMock.AssertNotCalled(t, "CleanBatch", mock.Anything)
}

func TestDoUpdateRates_ApplyUpdatesError_Propagates(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	cacheMock := new(MockRateUpdateCache)
	pending := []domain.PendingRateUpdate{
		{UpdateID: uuid.New(), PairID: 5, Base: "USD", Quote: "EUR"},
	}
	pairs := map[domain.RatePair]float64{
		{Base: "USD", Quote: "EUR"}: 1.01,
	}
	wantErr := errors.New("db fail")

	mockUpdatesRepo.On("ApplyUpdates", mock.Anything, mock.Anything).Return(wantErr).Once()

	count, err := doUpdateRates(context.Background(), pending, pairs, mockUpdatesRepo, cacheMock)

	require.Error(t, err)
	require.ErrorContains(t, err, "failed to update rates")
	require.Equal(t, 0, count)
	mockUpdatesRepo.AssertExpectations(t)
	cacheMock.AssertNotCalled(t, "CleanBatch", mock.Anything)
}

// --- UpdatePendingRates ---

func TestUpdatePendingRates_GetPendingError(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockClient := new(MockRateClient)
	cacheMock := new(MockRateUpdateCache)
	wantErr := errors.New("db unavailable")

	mockUpdatesRepo.On("GetPending", mock.Anything).Return(nil, wantErr).Once()

	err := UpdatePendingRates(context.Background(), "exec-1", mockUpdatesRepo, mockClient, cacheMock)

	require.Error(t, err)
	require.ErrorContains(t, err, "failed to get pending rates")
	mockUpdatesRepo.AssertExpectations(t)
	mockClient.AssertExpectations(t)
	cacheMock.AssertNotCalled(t, "CleanBatch", mock.Anything)
}

func TestUpdatePendingRates_NoPending(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockClient := new(MockRateClient)
	cacheMock := new(MockRateUpdateCache)

	mockUpdatesRepo.On("GetPending", mock.Anything).Return([]domain.PendingRateUpdate{}, nil).Once()

	err := UpdatePendingRates(context.Background(), "exec-2", mockUpdatesRepo, mockClient, cacheMock)

	require.NoError(t, err)
	mockUpdatesRepo.AssertExpectations(t)
	mockUpdatesRepo.AssertNotCalled(t, "ApplyUpdates", mock.Anything, mock.Anything)
	mockClient.AssertExpectations(t)
	cacheMock.AssertNotCalled(t, "CleanBatch", mock.Anything)
}

func TestUpdatePendingRates_Success_AppliesValues(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockClient := new(MockRateClient)
	cacheMock := new(MockRateUpdateCache)

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

	expectedPairs := []domain.RatePair{
		{Base: "USD", Quote: "EUR"},
		{Base: "EUR", Quote: "PLN"},
	}
	cacheMock.On("CleanBatch", mock.MatchedBy(func(pairs []domain.RatePair) bool {
		return assert.ElementsMatch(t, expectedPairs, pairs)
	})).Return().Once()

	err := UpdatePendingRates(context.Background(), "exec-3", mockUpdatesRepo, mockClient, cacheMock)

	require.NoError(t, err)
	mockUpdatesRepo.AssertExpectations(t)
	mockClient.AssertExpectations(t)
	cacheMock.AssertExpectations(t)
}

func TestDoUpdateRates_CleansCacheForUpdatedPairs(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	cacheMock := new(MockRateUpdateCache)

	pending := []domain.PendingRateUpdate{
		{UpdateID: uuid.New(), PairID: 1, Base: "USD", Quote: "EUR"},
		{UpdateID: uuid.New(), PairID: 2, Base: "EUR", Quote: "USD"},
	}
	pairs := map[domain.RatePair]float64{
		{Base: "USD", Quote: "EUR"}: 1.2,
	}

	mockUpdatesRepo.On("ApplyUpdates", mock.Anything, mock.Anything).Return(nil).Once()
	expectedPairs := []domain.RatePair{
		{Base: "USD", Quote: "EUR"},
		{Base: "EUR", Quote: "USD"},
	}
	cacheMock.On("CleanBatch", mock.MatchedBy(func(pairs []domain.RatePair) bool {
		return assert.ElementsMatch(t, expectedPairs, pairs)
	})).Return().Once()

	count, err := doUpdateRates(context.Background(), pending, pairs, mockUpdatesRepo, cacheMock)

	require.NoError(t, err)
	require.Equal(t, 2, count)
	mockUpdatesRepo.AssertExpectations(t)
	cacheMock.AssertExpectations(t)
}

func TestUpdatePendingRates_ApplyUpdatesError_Propagates(t *testing.T) {
	mockUpdatesRepo := new(MockRateUpdateRepository)
	mockClient := new(MockRateClient)
	cacheMock := new(MockRateUpdateCache)

	p1 := domain.PendingRateUpdate{UpdateID: uuid.New(), PairID: 1, Base: "USD", Quote: "EUR"}
	mockUpdatesRepo.On("GetPending", mock.Anything).Return([]domain.PendingRateUpdate{p1}, nil).Once()

	mockClient.On("GetExchangeRates", mock.Anything, "USD").Return(map[string]float64{"EUR": 1.11}, nil).Once()

	wantErr := errors.New("apply failed")
	mockUpdatesRepo.On("ApplyUpdates", mock.Anything, mock.Anything).Return(wantErr).Once()

	err := UpdatePendingRates(context.Background(), "exec-4", mockUpdatesRepo, mockClient, cacheMock)

	require.Error(t, err)
	require.ErrorContains(t, err, "failed to update rates")
	mockUpdatesRepo.AssertExpectations(t)
	mockClient.AssertExpectations(t)
	cacheMock.AssertNotCalled(t, "CleanBatch", mock.Anything)
}
