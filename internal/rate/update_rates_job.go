package rate

import (
	"context"
	"fmt"
	"fxrates/internal/adapters"
	"fxrates/internal/domain"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const numWorkers = 5
const perRequestTimeout = 5 * time.Second

type rateUpdate struct {
	Pair  domain.RatePair
	Value float64
}

// UpdatePendingRates updates rates in database with values from external API
func UpdatePendingRates(ctx context.Context, execID string, rateUpdateRepo adapters.RateUpdateRepository, rateClient adapters.RateClient, cache adapters.RateUpdateCache) error {
	// STEP 1: getting pending rate updates from DB
	pending, err := rateUpdateRepo.GetPending(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending rates: %w", err)
	}

	if len(pending) == 0 {
		logrus.Infof("Nothing to update this time; execID: %s", execID)
		return nil
	}

	logrus.Infof("%d pending rates were found, start updating; execID: %s", len(pending), execID)

	// STEP 2: collecting found pending rates into map like this:
	// {
	//	{ Base: "USD", Quote: "EUR" } -> -1.0,
	//	{ Base: "USD", Quote: "MXN" } -> -1.0,
	//	{ Base: "MXN", Quote: "EUR" } -> -1.0,
	//	...
	// }
	// ! NOTE 1: all the values are set as default -1.0
	// ! NOTE 2: map doesn't contain reversed pairs (for example if "USD/EUR" presents, then "EUR/USD" will not)
	// ! NOTE 3: this map is read by multiple threads down below, but updated ONLY BY THE SINGLE MAIN THREAD
	pairsMap := getUniquePairs(pending)

	// STEP 3: processing pairs in parallel using workers pool
	processInParallel(ctx, rateClient, pairsMap)

	// STEP 4: actually updating values in DB and clean cache
	countUpdated, err := doUpdateRates(ctx, pending, pairsMap, rateUpdateRepo, cache)
	if err != nil {
		return err
	}

	logrus.Infof("%d pending rates were successfully updated; execID %s", countUpdated, execID)
	return nil
}

func getUniquePairs(pending []domain.PendingRateUpdate) map[domain.RatePair]float64 {
	pairsMap := make(map[domain.RatePair]float64, len(pending))
	for _, rate := range pending {
		pair := domain.RatePair{Base: rate.Base, Quote: rate.Quote}
		if _, ok := pairsMap[pair.Reversed()]; ok {
			continue // skipping "EUR/USD" if "USD/EUR" presents
		}
		pairsMap[pair] = -1 // add pair with default value
	}
	return pairsMap
}

// processInParallel runs workers, which fetch rates from external API
func processInParallel(ctx context.Context, rateClient adapters.RateClient, pairsMap map[domain.RatePair]float64) {
	// STEP 1: extracting unique "bases", because pairsMap can contain same "Base" values, for example "USD":
	// {
	// 	{ Base: "USD", Quote: "EUR" } -> -1.0,
	//	{ Base: "USD", Quote: "MXN" } -> -1.0
	//  ...
	// }
	// For such cases we should not make several requests as values can be computed! So let's extract only unique "bases"
	bases := getUniqueBases(pairsMap) // bases will look like: ["USD", "EUR", ...]

	// STEP 2: creating workQueue for parallel execution and then using it for parallel http requests
	workQueue := make(chan string, len(bases))
	for _, base := range bases {
		workQueue <- base // workQueue simply stores codes ("USD", "EUR", etc)
	}
	close(workQueue)

	// STEP 3: running workers in parallel. Each worker puts its results into channel
	updatesCh := make(chan rateUpdate, len(pairsMap))

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, workerID, workQueue, rateClient, pairsMap, updatesCh)
		}(i)
	}

	wg.Wait()
	close(updatesCh)

	// STEP 4: after all workers finished their jobs, update values in pairsMap
	for upd := range updatesCh {
		pairsMap[upd.Pair] = upd.Value // key presence checked before passing to channel
	}
}

func getUniqueBases(pairsMap map[domain.RatePair]float64) []string {
	baseSet := make(map[string]struct{})
	for p := range pairsMap {
		baseSet[p.Base] = struct{}{}
	}

	bases := make([]string, 0, len(baseSet))
	for base := range baseSet {
		bases = append(bases, base)
	}
	return bases
}

func runWorker(ctx context.Context, workerID int, workQueue <-chan string, rateClient adapters.RateClient, pairsMap map[domain.RatePair]float64, updatesCh chan<- rateUpdate) {
	for {
		select {
		case <-ctx.Done():
			return
		case base, ok := <-workQueue:
			if !ok {
				return
			}
			processBase(ctx, workerID, base, rateClient, pairsMap, updatesCh)
		}
	}
}

// processBase fetches new values from external API and pushes matching pairs to the updates channel
func processBase(ctx context.Context, workerID int, base string, rateClient adapters.RateClient, pairsMap map[domain.RatePair]float64, updatesCh chan<- rateUpdate) {
	reqCtx, cancel := context.WithTimeout(ctx, perRequestTimeout)
	defer cancel()
	// STEP 1: make external API request
	// We are using context with timeout as we better interrupt request and process "Base" on the next scheduler job rather than wait!
	// After successful request, ratesMap will look like this:
	// {
	//		"MXN": 1.234,
	//		"EUR": 1.431,
	//      ...
	// }
	ratesMap, err := rateClient.GetExchangeRates(reqCtx, base)
	if err != nil {
		logrus.Warnf("Base '%s' wasn't processed by Worker %d as external api call returned error: %s", base, workerID, err)
		return
	}

	// STEP 2: iterating over ratesMap from response, find all pairs that present in pairsMap and put them into channel with updated values
	// NOTE 1 !!! to avoid confusion:
	// - ratesMap is what we get from API response
	// - pairsMap is our map where we store pairs and their values
	for quote, v := range ratesMap {
		p := domain.RatePair{Base: base, Quote: quote}
		if _, ok := pairsMap[p]; ok {
			updatesCh <- rateUpdate{Pair: p, Value: v}
		}
	}
}

// doUpdateRates actually updates rates in DB and cleans cache
func doUpdateRates(ctx context.Context, pending []domain.PendingRateUpdate, pairsMap map[domain.RatePair]float64, rateUpdatesRepo adapters.RateUpdateRepository, cache adapters.RateUpdateCache) (int, error) {
	// STEP 1: for all pending rates we:
	// - build a list of AppliedRateUpdate, which will be updated in DB
	// - build a list of RatePairs, which will be cleaned from cache
	updatesToApply := make([]domain.AppliedRateUpdate, 0, len(pending))
	updatedPairs := make([]domain.RatePair, 0, len(pending))

	for _, pr := range pending {
		var value float64
		pair := domain.RatePair{Base: pr.Base, Quote: pr.Quote}

		if v, ok := pairsMap[pair]; ok && v > 0 {
			value = v
		} else if v, ok = pairsMap[pair.Reversed()]; ok && v > 0 {
			// check if reversed pair in pairsMap and compute the value
			value = 1 / v
		} else {
			// this can happen when some workers failed to fetch rates from external api
			logrus.Warnf("Skipping update for '%s', it'll be processed next time", pr.Base+"/"+pr.Quote)
			continue
		}

		updatesToApply = append(updatesToApply, domain.AppliedRateUpdate{UpdateID: pr.UpdateID, PairID: pr.PairID, Value: value})
		updatedPairs = append(updatedPairs, domain.RatePair{Base: pr.Base, Quote: pr.Quote})
	}

	if len(updatesToApply) == 0 {
		return 0, nil
	}

	// STEP 2: applying updates in DB and clean cache
	err := rateUpdatesRepo.ApplyUpdates(ctx, updatesToApply)
	if err != nil {
		return 0, fmt.Errorf("failed to update rates: %w", err)
	}
	// Potentially before CleanBatch called, some other thread can access old cache inside ScheduleUpdate (service.go).
	// This isn't a problem as user will get fresh data on the next request
	cache.CleanBatch(updatedPairs)
	return len(updatedPairs), nil
}
