package rate

import (
	"context"
	"fxrates/internal/adapters"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Scheduler struct {
	rateUpdateRepo adapters.RateUpdateRepository
	rateClient     adapters.RateClient
	cache          adapters.RateUpdateCache
	// -----
	sched                  gocron.Scheduler
	updateRatesJobDuration time.Duration
}

func (s *Scheduler) Start(ctx context.Context) error {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return err
	}
	s.sched = scheduler

	job := func(jobCtx context.Context) {
		execID := uuid.NewString()
		updErr := UpdatePendingRates(jobCtx, execID, s.rateUpdateRepo, s.rateClient, s.cache)
		if updErr != nil {
			logrus.Errorf("Update pending rates job %s failed: %v", execID, updErr)
		}
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(s.updateRatesJobDuration),
		gocron.NewTask(job),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		return err
	}

	scheduler.Start()

	// Stop scheduler when the provided context is canceled.
	go func() {
		<-ctx.Done()
		if sdErr := s.Shutdown(); sdErr != nil {
			logrus.Errorf("scheduler shutdown error: %v", sdErr)
		}
	}()
	return nil
}

func (s *Scheduler) Shutdown() error {
	if s.sched == nil {
		return nil
	}
	err := s.sched.Shutdown()
	s.sched = nil
	return err
}

func NewScheduler(
	rateUpdatesRepo adapters.RateUpdateRepository,
	rateClient adapters.RateClient,
	cache adapters.RateUpdateCache,
	updateRatesJobDuration time.Duration,
) *Scheduler {
	if updateRatesJobDuration <= 0 {
		updateRatesJobDuration = 30 * time.Second
	}
	return &Scheduler{
		rateUpdateRepo:         rateUpdatesRepo,
		rateClient:             rateClient,
		cache:                  cache,
		updateRatesJobDuration: updateRatesJobDuration,
	}
}
