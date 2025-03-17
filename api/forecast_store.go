package api

import (
	"sync"
	"time"

	"weather-service/models"
)

// ForecastStore holds the latest forecast data organized by location and provider
type ForecastStore struct {
	data  map[string]map[string]models.ForecastData // key is location, then provider
	mutex sync.RWMutex
}

// NewForecastStore creates a new in-memory forecast data store
func NewForecastStore() *ForecastStore {
	return &ForecastStore{
		data: make(map[string]map[string]models.ForecastData),
	}
}

// UpdateForecast adds or updates forecast data for a location
func (s *ForecastStore) UpdateForecast(data models.ForecastData) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	location := data.Location
	provider := data.Provider

	// Check if we already have data for this location
	if _, exists := s.data[location]; !exists {
		s.data[location] = make(map[string]models.ForecastData)
	}

	// Store the forecast data
	s.data[location][provider] = data
}

// GetForecastByLocation retrieves all forecast data for a specific location
func (s *ForecastStore) GetForecastByLocation(location string) ([]models.ForecastData, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	providerMap, exists := s.data[location]
	if !exists {
		return nil, false
	}

	forecasts := make([]models.ForecastData, 0, len(providerMap))
	for _, forecast := range providerMap {
		forecasts = append(forecasts, forecast)
	}

	return forecasts, true
}

// GetForecastByProvider retrieves forecast data for a specific location and provider
func (s *ForecastStore) GetForecastByProvider(location, provider string) (models.ForecastData, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	providerMap, exists := s.data[location]
	if !exists {
		return models.ForecastData{}, false
	}

	forecast, exists := providerMap[provider]
	return forecast, exists
}

// GetAllForecastLocations returns a list of all locations with forecast data
func (s *ForecastStore) GetAllForecastLocations() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	locations := make([]string, 0, len(s.data))
	for loc := range s.data {
		locations = append(locations, loc)
	}
	return locations
}

// PruneOldForecasts removes forecasts older than the specified duration
func (s *ForecastStore) PruneOldForecasts(maxAge time.Duration) int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	prunedCount := 0

	for location, providers := range s.data {
		for provider, forecast := range providers {
			if forecast.Updated.Before(cutoff) {
				delete(s.data[location], provider)
				prunedCount++
			}
		}

		// If location has no more forecasts, remove it
		if len(s.data[location]) == 0 {
			delete(s.data, location)
		}
	}

	return prunedCount
}
