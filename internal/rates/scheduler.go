package rates

import (
	"context"
	"log"
	"time"

	"fxrates/internal/adapters/ratesapi"

	"github.com/go-co-op/gocron/v2"
)

type Scheduler struct {
	RateUpdatesRepo UpdatesRepository
	Client          *ratesapi.Client
}

func (s *Scheduler) CreateAndRun() {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Println(err)
		// todo: handle error
	}
	// defer func() { _ = scheduler.Shutdown() }()

	job := func(ctx context.Context) {
		UpdateRates(ctx, s.RateUpdatesRepo, s.Client)
	}

	_, err = scheduler.NewJob(
		gocron.DurationJob(5*time.Second),
		gocron.NewTask(job),
	)

	if err != nil {
		log.Println(err)
		// todo: handle error
	}

	log.Println("Starting scheduler")
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
