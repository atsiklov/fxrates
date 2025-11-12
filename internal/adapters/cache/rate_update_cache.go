package cache

import (
	"fmt"
	"fxrates/internal/domain"

	"github.com/dgraph-io/ristretto"
	"github.com/google/uuid"
)

type RistrettoRateUpdateCache struct {
	cache *ristretto.Cache
}

func NewRateUpdateCache(maxItems int64) (*RistrettoRateUpdateCache, error) {
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 10 * maxItems,
		MaxCost:     maxItems,
		BufferItems: 64,
	})
	if err != nil {
		return nil, fmt.Errorf("create rate update cache failed: %w", err)
	}
	return &RistrettoRateUpdateCache{cache: c}, nil
}

func (c *RistrettoRateUpdateCache) Get(pair domain.RatePair) (uuid.UUID, bool) {
	if v, ok := c.cache.Get(toKey(pair)); ok {
		id, ok := v.(uuid.UUID)
		return id, ok
	}
	return uuid.Nil, false
}

func (c *RistrettoRateUpdateCache) Set(pair domain.RatePair, id uuid.UUID) {
	c.cache.Set(toKey(pair), id, 1)
}

func (c *RistrettoRateUpdateCache) CleanBatch(pairs []domain.RatePair) {
	for _, pair := range pairs {
		c.cache.Del(toKey(pair))
	}
}

func (c *RistrettoRateUpdateCache) Close() { c.cache.Close() }

func toKey(p domain.RatePair) string { return p.Base + ":" + p.Quote }
