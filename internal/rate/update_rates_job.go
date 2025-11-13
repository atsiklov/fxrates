package rate

import (
	"context"
	"fmt"
	"fxrates/internal/adapters"
	"fxrates/internal/domain"
	"maps"
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

	// STEP 2: collecting found rates into a set like this:
	// {
	//	{ Base: "USD", Quote: "EUR" },
	//	{ Base: "USD", Quote: "MXN" },
	//	{ Base: "MXN", Quote: "EUR" },
	//	...
	// }
	// ! NOTE: set doesn't contain reversed pairs (for example if "USD/EUR" presents, then "EUR/USD" will not)
	pairSet := getUniquePairs(pending)

	// STEP 3: processing set in parallel using workers pool. The result is a map of pairs with values
	pairValueMap := processInParallel(ctx, rateClient, pairSet)

	// STEP 4: actually updating values in DB, then cleaning cache
	countUpdated, err := doUpdateRates(ctx, pending, pairValueMap, rateUpdateRepo, cache)
	if err != nil {
		return err
	}

	logrus.Infof("%d pending rates were successfully updated; execID %s", countUpdated, execID)
	return nil
}

func getUniquePairs(pending []domain.PendingRateUpdate) map[domain.RatePair]struct{} {
	pairSet := make(map[domain.RatePair]struct{}, len(pending))
	for _, rate := range pending {
		pair := domain.RatePair{Base: rate.Base, Quote: rate.Quote}
		if _, ok := pairSet[pair.Reversed()]; ok {
			continue // skipping "EUR/USD" if "USD/EUR" presents
		}
		pairSet[pair] = struct{}{}
	}
	return pairSet
}

// processInParallel runs workers, which fetch rates from external API
func processInParallel(ctx context.Context, rateClient adapters.RateClient, pairs map[domain.RatePair]struct{}) map[domain.RatePair]float64 {
	// STEP 1: extracting unique "bases"
	// Pairs can contain same base values, for example "USD/EUR and "USD/MXN", we should not
	// make several requests for the same currency! So let's extract only unique "bases"
	bases := getUniqueBases(pairs) // bases is a set like: {"USD" -> {}, "EUR" -> {}, ...}

	// STEP 2: creating workQueue for parallel execution and then using it for parallel http requests
	workQueue := make(chan string, len(bases))
	for base := range maps.Keys(bases) {
		workQueue <- base // workQueue simply stores codes ("USD", "EUR", etc)
	}
	close(workQueue)

	// STEP 3: running workers in parallel. Each worker puts its results into channel
	updatesCh := make(chan rateUpdate, len(pairs))

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, workerID, workQueue, rateClient, pairs, updatesCh)
		}(i)
	}

	wg.Wait()
	close(updatesCh)

	// STEP 4: after all workers finished their jobs, creating a map containing pairs with values
	pairValueMap := make(map[domain.RatePair]float64, len(pairs))
	for upd := range updatesCh {
		pairValueMap[upd.Pair] = upd.Value
	}
	return pairValueMap
}

func getUniqueBases(pairs map[domain.RatePair]struct{}) map[string]struct{} {
	baseSet := make(map[string]struct{})
	for p := range pairs {
		baseSet[p.Base] = struct{}{}
	}
	return baseSet
}

func runWorker(ctx context.Context, workerID int, workQueue <-chan string, rateClient adapters.RateClient, pairs map[domain.RatePair]struct{}, updatesCh chan<- rateUpdate) {
	for {
		select {
		case <-ctx.Done():
			return
		case base, ok := <-workQueue:
			if !ok {
				return
			}
			processBase(ctx, workerID, base, rateClient, pairs, updatesCh)
		}
	}
}

// processBase fetches new values from external API and pushes matching pairs to the updates channel
func processBase(ctx context.Context, workerID int, base string, rateClient adapters.RateClient, pairs map[domain.RatePair]struct{}, updatesCh chan<- rateUpdate) {
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
	for quote, v := range ratesMap {
		p := domain.RatePair{Base: base, Quote: quote}
		if _, ok := pairs[p]; ok {
			updatesCh <- rateUpdate{Pair: p, Value: v}
		}
	}
}

// doUpdateRates actually updates rates in DB and cleans cache
func doUpdateRates(ctx context.Context, pending []domain.PendingRateUpdate, pairValueMap map[domain.RatePair]float64, rateUpdatesRepo adapters.RateUpdateRepository, cache adapters.RateUpdateCache) (int, error) {
	// STEP 1: for all pending rates we:
	// - build a list of AppliedRateUpdate, which will be updated in DB
	// - build a list of RatePairs, which will be cleaned from cache
	updatesToApply := make([]domain.AppliedRateUpdate, 0, len(pending))
	updatedPairs := make([]domain.RatePair, 0, len(pending))

	for _, pr := range pending {
		var value float64
		pair := domain.RatePair{Base: pr.Base, Quote: pr.Quote}

		if v, ok := pairValueMap[pair]; ok {
			value = v
		} else if v, ok = pairValueMap[pair.Reversed()]; ok {
			// check if reversed pair presents and compute the value
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
