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
const perRequestTimeout = 3 * time.Second

type pair struct {
	Base  string
	Quote string
}

// UpdatePendingRates updates rate in database using values from external API
func UpdatePendingRates(ctx context.Context, execID string, rateUpdatesRepo adapters.RateUpdatesRepository, rateClient adapters.RateClient) error {
	// step 1: get rate from DB that require update
	pending, err := rateUpdatesRepo.GetPending(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending rates: %w", err)
	}

	if len(pending) == 0 {
		logrus.Infof("Nothing to update this time; execID: %s", execID)
		return nil
	}

	logrus.Infof("%d pending rates were found, start updating; execID: %s", len(pending), execID)

	// step 2: collecting rate into map like this:
	// {
	//	key                           ->  value,
	//	{ Base: "USD", Quote: "EUR" } -> -1.000,
	//	{ Base: "USD", Quote: "MXN" } -> -1.000,
	//	{ Base: "MXN", Quote: "EUR" } -> -1.000,
	//	...
	// }
	// ! NOTE 1: all the values are set as default -1.000
	// ! NOTE 2: map doesn't contain reversed pairs (for example if "USD/EUR" presents, then "EUR/USD" will not)
	// !!! NOTE 3: this map will be our store which will be used to update values in rate
	pairsMap := getUniquePairs(pending)

	// step 3: process pairs in parallel using worker pool
	processInParallel(ctx, rateClient, pairsMap)

	// step 4:
	countUpdated, err := doUpdateRates(ctx, pending, pairsMap, rateUpdatesRepo)
	if err != nil {
		return err
	}

	logrus.Infof("%d pending rates were successfully updated; execID %s", countUpdated, execID)
	return nil
}

func getUniquePairs(pending []domain.PendingRate) map[pair]float64 {
	pairsMap := make(map[pair]float64, len(pending))
	for _, rate := range pending {
		reversedPair := pair{Base: rate.Quote, Quote: rate.Base}
		if _, ok := pairsMap[reversedPair]; ok {
			continue // Skipping "EUR/USD" if "USD/EUR" pair presents
		}
		pairsMap[pair{Base: rate.Base, Quote: rate.Quote}] = -1.000 // add pair with default value
	}
	return pairsMap
}

// processInParallel runs parallel workers, which fetch rate from external API and replace values in pairs map
func processInParallel(ctx context.Context, rateClient adapters.RateClient, pairsMap map[pair]float64) {
	// Extracting unique "bases"
	// Explanation: pairsMap can contain same "base" values, for example:
	// {
	// 	{ Base: "USD", Quote: "EUR" } -> -1.000,
	//	{ Base: "USD", Quote: "MXN" } -> -1.000
	//  ...
	// }
	// These should not be separate requests. So let's extract only unique "bases" in order to optimize requests count
	bases := getUniqueBases(pairsMap) // bases will look like: ["USD", "EUR", ...]

	// Creating workQueue for parallel execution and then using it for parallel http requests
	workQueue := make(chan string, len(bases))
	for _, base := range bases {
		workQueue <- base // workQueue simply stores codes ("USD", "EUR", etc)
	}
	close(workQueue)

	var wg sync.WaitGroup
	var mu sync.Mutex // Using mutex when updating "pairsMap" concurrently
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, workerID, workQueue, rateClient, pairsMap, &mu)
		}(i)
	}

	// Waiting for all workers to finish
	wg.Wait()
}

func getUniqueBases(pairsMap map[pair]float64) []string {
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

func runWorker(ctx context.Context, workerID int, workQueue <-chan string, rateClient adapters.RateClient, pairsMap map[pair]float64, mu *sync.Mutex) {
	for {
		select {
		case <-ctx.Done():
			return
		case base, ok := <-workQueue:
			if !ok {
				return
			}
			processBase(ctx, workerID, base, rateClient, pairsMap, mu)
		}
	}
}

// processBase fetches new values from external API and replaces values in pairs map
func processBase(ctx context.Context, workerID int, base string, rateClient adapters.RateClient, pairsMap map[pair]float64, mu *sync.Mutex) {
	reqCtx, cancel := context.WithTimeout(ctx, perRequestTimeout)
	defer cancel()
	// Fetching rates for the specified "base" from external API. Using context with timeout as we better interrupt
	// request and process "base" on the next scheduler job rather than wait
	// ratesMap will look like this:
	// {
	//		"MXN": 1.234,
	//		"EUR": 1.431
	// }
	ratesMap, err := rateClient.GetExchangeRates(reqCtx, base)
	if err != nil {
		logrus.Warnf("Base '%s' wasn't processed by Worker %d as external api call returned error: %s", base, workerID, err)
		return
	}

	// Updating values (which are currently default -1.000) in pairs map
	// ---
	// Basically "base" is always fixed, so we iterate over ratesMap and on each iteration we
	// create pair like {"USD", "<other code>"} and check if it presents in pairs map. If it does, replacing value
	for quote, v := range ratesMap {
		p := pair{Base: base, Quote: quote}
		if _, ok := pairsMap[p]; ok {
			mu.Lock() // lock the entire map, hope this fine for test project :)
			pairsMap[p] = v
			mu.Unlock()
		}
	}
}

// doUpdateRates actually updates values in our domain rate using pairs map
// - for each pending rate find corresponding pair and build applied rate with received value
// - if desired pair absents, taking reversed pair and compute the value
func doUpdateRates(ctx context.Context, pending []domain.PendingRate, pairsMap map[pair]float64, rateUpdatesRepo adapters.RateUpdatesRepository) (int, error) {
	applied := make([]domain.AppliedRate, 0, len(pending))

	for _, pr := range pending {
		var value float64

		if v, ok := pairsMap[pair{Base: pr.Base, Quote: pr.Quote}]; ok && v > 0 {
			value = v
		} else if v, ok = pairsMap[pair{Base: pr.Quote, Quote: pr.Base}]; ok && v > 0 {
			value = 1 / v
		} else {
			// this can happen when some workers failed to fetch rates from external api
			logrus.Warnf("Skipping update for '%s', it'll be processed next time", pr.Base+"/"+pr.Quote)
			continue
		}

		applied = append(applied, domain.AppliedRate{
			PairID: pr.PairID,
			Base:   pr.Base,
			Quote:  pr.Quote,
			Value:  value,
			// UpdatedAt - updates at db level
		})
	}

	if len(applied) == 0 {
		return 0, nil
	}

	err := rateUpdatesRepo.SaveApplied(ctx, applied)
	if err != nil {
		return 0, fmt.Errorf("failed to update rates: %w", err)
	}

	return len(applied), nil
}
