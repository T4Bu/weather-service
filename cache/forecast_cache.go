package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"weather-service/datasource"
	"weather-service/models"
)

// CachedForecastSource wraps a ForecastSource and adds caching functionality
type CachedForecastSource struct {
	source         datasource.ForecastSource
	cache          map[string]forecastCacheEntry // key is location:days
	mutex          sync.RWMutex
	cacheDuration  time.Duration
	cacheHitCount  int
	cacheMissCount int
}

// forecastCacheEntry represents a cached forecast with its timestamp
type forecastCacheEntry struct {
	Data      models.ForecastData
	Timestamp time.Time
}

// NewCachedForecastSource creates a new cached wrapper around a forecast source
func NewCachedForecastSource(source datasource.ForecastSource, cacheDuration time.Duration) *CachedForecastSource {
	return &CachedForecastSource{
		source:        source,
		cache:         make(map[string]forecastCacheEntry),
		cacheDuration: cacheDuration,
	}
}

// Name returns the name of the underlying forecast source with [Cached] prefix
func (c *CachedForecastSource) Name() string {
	return c.source.Name() + " [Cached]"
}

// FetchForecast fetches forecast data, using cache when available
func (c *CachedForecastSource) FetchForecast(ctx context.Context, location string, days int) (models.ForecastData, error) {
	// Create a cache key that combines location and days
	cacheKey := fmt.Sprintf("%s:%d", location, days)

	// First check if we have this forecast in the cache
	c.mutex.RLock()
	entry, found := c.cache[cacheKey]
	c.mutex.RUnlock()

	// If found and not expired, return the cached forecast
	if found && time.Since(entry.Timestamp) < c.cacheDuration {
		c.mutex.Lock()
		c.cacheHitCount++
		c.mutex.Unlock()

		fmt.Printf("Forecast Cache HIT for %s (days=%d) from %s (age: %s)\n",
			location, days, c.source.Name(), time.Since(entry.Timestamp).Round(time.Second))

		return entry.Data, nil
	}

	// Cache miss or expired, fetch fresh forecast
	c.mutex.Lock()
	c.cacheMissCount++
	c.mutex.Unlock()

	fmt.Printf("Forecast Cache MISS for %s (days=%d) from %s, fetching fresh data...\n",
		location, days, c.source.Name())

	forecast, err := c.source.FetchForecast(ctx, location, days)
	if err != nil {
		return models.ForecastData{}, err
	}

	// Store in cache
	c.mutex.Lock()
	c.cache[cacheKey] = forecastCacheEntry{
		Data:      forecast,
		Timestamp: time.Now(),
	}
	c.mutex.Unlock()

	return forecast, nil
}

// CacheStats returns statistics about cache hits and misses
func (c *CachedForecastSource) CacheStats() (hits, misses int) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.cacheHitCount, c.cacheMissCount
}

// Ensure CachedForecastSource implements ForecastSource
var _ datasource.ForecastSource = (*CachedForecastSource)(nil)
