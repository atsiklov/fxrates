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
	rateUpdatesRepo adapters.RateUpdateRepository
	rateClient      adapters.RateClient
	// -----
	sched       gocron.Scheduler
	jobDuration time.Duration
}

func (s *Scheduler) Start(ctx context.Context) error {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return err
	}
	s.sched = scheduler

	job := func(jobCtx context.Context) {
		execID := uuid.NewString()
		updErr := UpdatePendingRates(jobCtx, execID, s.rateUpdatesRepo, s.rateClient)
		if updErr != nil {
			logrus.Errorf("ApplyUpdates pending rates job %s failed: %v", execID, updErr)
		}
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(s.jobDuration),
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
			logrus.Errorf("Scheduler shutdown error: %v", sdErr)
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

func NewScheduler(rateUpdatesRepo adapters.RateUpdateRepository, rateClient adapters.RateClient, jobDuration time.Duration) *Scheduler {
	if jobDuration <= 0 {
		jobDuration = 30 * time.Second
	}
	return &Scheduler{rateUpdatesRepo: rateUpdatesRepo, rateClient: rateClient, jobDuration: jobDuration}
}
