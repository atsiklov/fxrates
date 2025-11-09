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
	rateUpdatesRepo adapters.RateUpdatesRepository
	rateClient      adapters.RateClient
	// -----
	sched gocron.Scheduler
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
			logrus.Errorf("Update pending rates job %s failed: %v", execID, updErr)
		}
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(10*time.Second),
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

func NewScheduler(rateUpdatesRepo adapters.RateUpdatesRepository, rateClient adapters.RateClient) *Scheduler {
	return &Scheduler{rateUpdatesRepo: rateUpdatesRepo, rateClient: rateClient}
}
