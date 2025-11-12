package rate

import (
	"context"
	"testing"
	"time"

	"fxrates/internal/domain"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewScheduler_Constructs(t *testing.T) {
	s := NewScheduler(new(MockRateUpdateRepository), new(MockRateClient), nil, 10*time.Second)
	require.NotNil(t, s)
	require.Nil(t, s.sched)
}

func TestScheduler_Shutdown_NoScheduler_ReturnsNil(t *testing.T) {
	s := NewScheduler(new(MockRateUpdateRepository), new(MockRateClient), nil, 10*time.Second)
	err := s.Shutdown()
	require.NoError(t, err)
	require.Nil(t, s.sched)
}

func TestScheduler_Start_And_ContextCancel_ShutsDown(t *testing.T) {
	s := NewScheduler(new(MockRateUpdateRepository), new(MockRateClient), nil, 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())

	// Start scheduler
	require.NoError(t, s.Start(ctx))
	require.NotNil(t, s.sched)

	// Cancel and ensure Shutdown is called by goroutine
	cancel()

	// Wait until s.sched becomes nil (Shutdown sets it to nil)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.sched == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	require.Nil(t, s.sched, "expected scheduler to be shutdown after ctx cancel")
}

func TestScheduler_Shutdown_AfterStart_Idempotent(t *testing.T) {
	repo := new(MockRateUpdateRepository)
	repo.On("GetPending", mock.Anything).Return([]domain.PendingRateUpdate{}, nil).Maybe()
	s := NewScheduler(repo, new(MockRateClient), nil, 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, s.Start(ctx))
	require.NotNil(t, s.sched)

	// First shutdown should stop scheduler and set field to nil
	require.NoError(t, s.Shutdown())
	require.Nil(t, s.sched)

	// Second shutdown should be a no-op and return nil
	require.NoError(t, s.Shutdown())
}

func TestNewScheduler_UsesProvidedInterval(t *testing.T) {
	s := NewScheduler(new(MockRateUpdateRepository), new(MockRateClient), nil, 42*time.Second)
	require.Equal(t, 42*time.Second, s.updateRatesJobDuration)
}

func TestNewScheduler_DefaultsIntervalWhenInvalid(t *testing.T) {
	s := NewScheduler(new(MockRateUpdateRepository), new(MockRateClient), nil, 0)
	require.Equal(t, 30*time.Second, s.updateRatesJobDuration)
}
