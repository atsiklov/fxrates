package cache

import (
	"testing"

	"fxrates/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRateUpdateCache_SetAndGet(t *testing.T) {
	c, err := NewRateUpdateCache(128)
	require.NoError(t, err)
	defer c.Close()

	pair := domain.RatePair{Base: "USD", Quote: "EUR"}
	updateID := uuid.New()

	c.Set(pair, updateID)
	c.cache.Wait()

	got, ok := c.Get(pair)
	require.True(t, ok)
	require.Equal(t, updateID, got)
}

func TestRateUpdateCache_GetMissWhenEmpty(t *testing.T) {
	c, err := NewRateUpdateCache(64)
	require.NoError(t, err)
	defer c.Close()

	id, ok := c.Get(domain.RatePair{Base: "EUR", Quote: "USD"})
	require.False(t, ok)
	require.Equal(t, uuid.Nil, id)
}

func TestRateUpdateCache_CleanBatchEvictsOnlySpecifiedPairs(t *testing.T) {
	c, err := NewRateUpdateCache(256)
	require.NoError(t, err)
	defer c.Close()

	usdeur := domain.RatePair{Base: "USD", Quote: "EUR"}
	eurusd := domain.RatePair{Base: "EUR", Quote: "USD"}
	usdjpy := domain.RatePair{Base: "USD", Quote: "JPY"}

	c.Set(usdeur, uuid.New())
	c.Set(eurusd, uuid.New())
	keepID := uuid.New()
	c.Set(usdjpy, keepID)
	c.cache.Wait()

	c.CleanBatch([]domain.RatePair{usdeur, eurusd})

	_, ok := c.Get(usdeur)
	require.False(t, ok)
	_, ok = c.Get(eurusd)
	require.False(t, ok)

	id, ok := c.Get(usdjpy)
	require.True(t, ok)
	require.Equal(t, keepID, id)
}
