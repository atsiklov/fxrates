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
}

func (s *Scheduler) CreateAndRun() {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		logrus.Fatalf("Error creating scheduler: %s", err) // todo: handle
	}
	// defer func() { _ = scheduler.Shutdown() }()

	job := func(ctx context.Context) {
		execID := uuid.NewString()
		err := UpdateRates(ctx, execID, s.rateUpdatesRepo, s.rateClient)
		if err != nil {
			logrus.Error("Internal job error") // todo: handle error
		}
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(5*time.Second),
		gocron.NewTask(job),
	)

	if err != nil {
		logrus.Fatalf("Error creating job: %s", err) // todo: handle
	}

	scheduler.Start()

	// todo: завершение по сигналу
	// sig := make(chan os.Signal, 1)
	// signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	// <-sig
	//
	// todo: корректная остановка
	// if err := s.Shutdown(); err != nil {
	// 	log.Fatal(err)
	// }
}

func NewScheduler(rateUpdatesRepo adapters.RateUpdatesRepository, rateClient adapters.RateClient) *Scheduler {
	return &Scheduler{rateUpdatesRepo: rateUpdatesRepo, rateClient: rateClient}
}
