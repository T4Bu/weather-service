package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"weather-service/datasource"
	"weather-service/models"
)

// CachedDataSource wraps a DataSource and adds caching functionality
type CachedDataSource struct {
	source         datasource.DataSource
	cache          map[string]cacheEntry
	mutex          sync.RWMutex
	cacheDuration  time.Duration
	cacheHitCount  int
	cacheMissCount int
}

// cacheEntry represents a cached weather data item with its timestamp
type cacheEntry struct {
	Data      models.WeatherData
	Timestamp time.Time
}

// NewCachedDataSource creates a new cached wrapper around a data source
func NewCachedDataSource(source datasource.DataSource, cacheDuration time.Duration) *CachedDataSource {
	return &CachedDataSource{
		source:        source,
		cache:         make(map[string]cacheEntry),
		cacheDuration: cacheDuration,
	}
}

// Name returns the name of the underlying data source with [Cached] prefix
func (c *CachedDataSource) Name() string {
	return c.source.Name() + " [Cached]"
}

// FetchWeatherData fetches weather data, using cache when available
func (c *CachedDataSource) FetchWeatherData(ctx context.Context, location string) (models.WeatherData, error) {
	// First check if we have this data in the cache
	c.mutex.RLock()
	entry, found := c.cache[location]
	c.mutex.RUnlock()

	// If found and not expired, return the cached data
	if found && time.Since(entry.Timestamp) < c.cacheDuration {
		c.mutex.Lock()
		c.cacheHitCount++
		c.mutex.Unlock()

		fmt.Printf("Cache HIT for %s from %s (age: %s)\n",
			location, c.source.Name(), time.Since(entry.Timestamp).Round(time.Second))

		return entry.Data, nil
	}

	// Cache miss or expired, fetch fresh data
	c.mutex.Lock()
	c.cacheMissCount++
	c.mutex.Unlock()

	fmt.Printf("Cache MISS for %s from %s, fetching fresh data...\n",
		location, c.source.Name())

	data, err := c.source.FetchWeatherData(ctx, location)
	if err != nil {
		return models.WeatherData{}, err
	}

	// Store in cache
	c.mutex.Lock()
	c.cache[location] = cacheEntry{
		Data:      data,
		Timestamp: time.Now(),
	}
	c.mutex.Unlock()

	return data, nil
}

// CacheStats returns statistics about cache hits and misses
func (c *CachedDataSource) CacheStats() (hits, misses int) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.cacheHitCount, c.cacheMissCount
}

// Ensure CachedDataSource implements the DataSource interface
var _ datasource.DataSource = (*CachedDataSource)(nil)
