package rates

import (
	"context"
	"fxrates/internal/adapters/ratesapi"
	"fxrates/internal/domain"
	"log"
	"sync"
)

const numWorkers = 5

type ratePair struct {
	Base  string
	Quote string
}

// UpdateRates updates rates in database using values from external API
func UpdateRates(ctx context.Context, repository UpdatesRepository, client *ratesapi.Client) {
	log.Println("Updating rates")
	// step 1: get rates from DB that require update
	rates, err := repository.GetPending(ctx)
	if err != nil {
		panic(err)
	}

	if len(rates) == 0 {
		return
	}

	// step 2: collecting rates into map like this:
	// {
	//	key                           ->  value,
	//	{ Base: "USD", Quote: "EUR" } -> -1.000,
	//	{ Base: "USD", Quote: "MXN" } -> -1.000,
	//	{ Base: "MXN", Quote: "EUR" } -> -1.000,
	//	...
	// }
	// ! NOTE 1: all the values are set as default -1.000
	// ! NOTE 2: map doesn't contain reversed pairs (for example if "USD/EUR" presents, then "EUR/USD" will not)
	// !!! NOTE 3: this map will be our store which will be used to update values in rates
	pairs := getUniquePairs(rates)

	// step 3: process pairs in parallel using worker pool
	processInParallel(ctx, client, pairs)

	// step 4:
	doUpdateRates(ctx, rates, pairs, repository)
}

func getUniquePairs(rates []domain.PendingRate) map[ratePair]float64 {
	pairs := make(map[ratePair]float64, len(rates))
	for _, rate := range rates {
		reversedPair := ratePair{Base: rate.Quote, Quote: rate.Base}
		if _, ok := pairs[reversedPair]; ok {
			continue // Skipping "EUR/USD" if "USD/EUR" pair presents
		}
		pairs[ratePair{Base: rate.Base, Quote: rate.Quote}] = -1.000 // add pair with default value
	}
	return pairs
}

// processInParallel runs parallel workers, which fetch rates from external API and replace values in pairs map
func processInParallel(ctx context.Context, client *ratesapi.Client, pairs map[ratePair]float64) {
	// Extracting unique "bases"
	// Explanation: pairs can contain same "base" values, for example:
	// {
	// 	{ Base: "USD", Quote: "EUR" } -> -1.000,
	//	{ Base: "USD", Quote: "MXN" } -> -1.000
	//  ...
	// }
	// These should not be separate requests. So let's extract only unique "bases" in order to optimize requests count
	bases := getUniqueBases(pairs) // bases will look like: ["USD", "EUR", ...]

	// Creating workQueue for parallel execution and then using it for parallel http requests
	workQueue := make(chan string, len(bases))
	for _, base := range bases {
		workQueue <- base // workQueue simply stores codes ("USD", "EUR", etc)
	}
	close(workQueue)

	var wg sync.WaitGroup
	var mu sync.Mutex // Using mutex when updating "pairs" concurrently
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, workerID, workQueue, client, pairs, &mu)
		}(i)
	}

	// Waiting for all workers to finish
	wg.Wait()
}

func getUniqueBases(pairs map[ratePair]float64) []string {
	baseSet := make(map[string]struct{})
	for pair, _ := range pairs {
		baseSet[pair.Base] = struct{}{}
	}

	bases := make([]string, 0, len(baseSet))
	for base := range baseSet {
		bases = append(bases, base)
	}
	return bases
}

func runWorker(ctx context.Context, workerID int, workQueue <-chan string, client *ratesapi.Client, pairs map[ratePair]float64, mu *sync.Mutex) {
	for base := range workQueue { // each worker takes base currency code from queue and process it
		processBase(ctx, workerID, base, client, pairs, mu)
	}
}

// processBase fetches new values from external API and replaces values in pairs map
func processBase(ctx context.Context, workerID int, base string, client *ratesapi.Client, pairs map[ratePair]float64, mu *sync.Mutex) {
	// Fetching rates for the specified "base" from external API
	// ratesMap looks like:
	// {
	//		"MXN": 1.234,
	//		"EUR": 1.431
	// }
	ratesMap, err := client.GetExchangeRate(ctx, base)
	if err != nil {
		panic(err) // todo: handle errors
	}

	// Updating values (which are currently default -1.000) in pairs map
	// Basically "base" is always fixed, so we iterate over ratesMap and on each iteration:
	// - create pair like {"USD", "<other code>"}
	// - check if it presents in pairs map
	// - if it does, replace default value
	for quote, v := range ratesMap {
		p := ratePair{Base: base, Quote: quote}
		if _, ok := pairs[p]; ok {
			mu.Lock() // todo: не лочить всю мапу
			pairs[p] = v
			mu.Unlock()
		}
	}
}

// doUpdateRates actually updates values in our domain rates using pairs map
// - for each pending rate find corresponding pair and build applied rate adding value
// - if desired pair absents, taking reversed pair and compute the value
func doUpdateRates(ctx context.Context, pendingRates []domain.PendingRate, pairs map[ratePair]float64, repo UpdatesRepository) error {
	appliedRates := make([]domain.AppliedRate, 0, len(pendingRates))

	for _, pr := range pendingRates {
		var value float64

		if v, ok := pairs[ratePair{Base: pr.Base, Quote: pr.Quote}]; ok && v > 0 {
			value = v
		} else if v, ok = pairs[ratePair{Base: pr.Quote, Quote: pr.Base}]; ok && v > 0 {
			value = 1 / v
		} else {
			continue // todo: log anomaly
		}

		appliedRates = append(appliedRates, domain.AppliedRate{
			PairID: pr.PairID,
			Base:   pr.Base,
			Quote:  pr.Quote,
			Value:  value,
			// UpdatedAt: now - updates at db level
		})
	}

	if len(appliedRates) == 0 {
		return nil
	}

	return repo.SaveApplied(ctx, appliedRates) // todo: handle errors
}
