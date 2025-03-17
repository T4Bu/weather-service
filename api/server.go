package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"weather-service/datasource"
	"weather-service/models"
)

// WeatherStore holds the latest weather data by location
type WeatherStore struct {
	data  map[string][]models.WeatherData // key is location, value is array of provider data
	mutex sync.RWMutex
}

// NewWeatherStore creates a new in-memory weather data store
func NewWeatherStore() *WeatherStore {
	return &WeatherStore{
		data: make(map[string][]models.WeatherData),
	}
}

// UpdateWeather adds or updates weather data for a location
func (s *WeatherStore) UpdateWeather(data models.WeatherData) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	location := data.Location

	// Check if we already have data for this location
	if _, exists := s.data[location]; !exists {
		s.data[location] = []models.WeatherData{}
	}

	// Find if we already have data from this provider
	found := false
	for i, existingData := range s.data[location] {
		if existingData.Provider == data.Provider {
			// Update existing entry
			s.data[location][i] = data
			found = true
			break
		}
	}

	// If no data from this provider exists, append it
	if !found {
		s.data[location] = append(s.data[location], data)
	}
}

// GetWeatherByLocation retrieves weather data for a specific location
func (s *WeatherStore) GetWeatherByLocation(location string) ([]models.WeatherData, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	data, exists := s.data[location]
	return data, exists
}

// GetAllLocations returns a list of all available locations
func (s *WeatherStore) GetAllLocations() []string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	locations := make([]string, 0, len(s.data))
	for loc := range s.data {
		locations = append(locations, loc)
	}
	return locations
}

// Server represents the API server
type Server struct {
	weatherStore    *WeatherStore
	forecastStore   *ForecastStore
	server          *http.Server
	forecastSources []datasource.ForecastSource
}

// NewServer creates a new API server
func NewServer(weatherStore *WeatherStore, forecastStore *ForecastStore, port int) *Server {
	mux := http.NewServeMux()

	server := &Server{
		weatherStore:  weatherStore,
		forecastStore: forecastStore,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}

	// Register handlers for current weather
	mux.HandleFunc("/api/weather/location/", server.handleGetWeatherByLocation)
	mux.HandleFunc("/api/weather/locations", server.handleGetAllLocations)

	// Register handlers for forecasts
	mux.HandleFunc("/api/forecast/location/", server.handleGetForecastByLocation)

	// Health check
	mux.HandleFunc("/api/health", server.handleHealthCheck)

	return server
}

// RegisterForecastSources adds forecast sources to the server
func (s *Server) RegisterForecastSources(sources []datasource.ForecastSource) {
	s.forecastSources = sources
}

// Start begins the API server
func (s *Server) Start() error {
	fmt.Printf("Starting API server on %s\n", s.server.Addr)
	return s.server.ListenAndServe()
}

// handleGetWeatherByLocation handles requests for weather data by location
func (s *Server) handleGetWeatherByLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract location from URL path
	path := r.URL.Path
	if len(path) <= len("/api/weather/location/") {
		http.Error(w, "Location not specified", http.StatusBadRequest)
		return
	}

	location := path[len("/api/weather/location/"):]
	data, exists := s.weatherStore.GetWeatherByLocation(location)

	w.Header().Set("Content-Type", "application/json")

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("No weather data found for location: %s", location),
		})
		return
	}

	response := map[string]interface{}{
		"location":  location,
		"data":      data,
		"timestamp": time.Now(),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleGetAllLocations returns a list of all locations with weather data
func (s *Server) handleGetAllLocations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	locations := s.weatherStore.GetAllLocations()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"locations": locations,
		"count":     len(locations),
	})
}

// handleGetForecastByLocation handles requests for forecast data by location
func (s *Server) handleGetForecastByLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract location from URL path
	path := r.URL.Path
	if len(path) <= len("/api/forecast/location/") {
		http.Error(w, "Location not specified", http.StatusBadRequest)
		return
	}

	// Extract days parameter from query string (default to 3 days)
	daysStr := r.URL.Query().Get("days")
	days := 3 // Default to 3 days
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
			if days > 5 {
				days = 5 // Cap at 5 days maximum
			}
		}
	}

	// Extract any path parameters after location
	pathParts := strings.Split(path[len("/api/forecast/location/"):], "/")
	location := pathParts[0]

	// Fetch from specific provider if specified
	var provider string
	if len(pathParts) > 1 && pathParts[1] != "" {
		provider = pathParts[1]
	}

	w.Header().Set("Content-Type", "application/json")

	// If a provider is specified, return just that provider's forecast
	if provider != "" {
		forecast, exists := s.forecastStore.GetForecastByProvider(location, provider)
		if !exists {
			// If we have forecast sources, try to fetch on-demand
			if len(s.forecastSources) > 0 && provider != "" {
				for _, source := range s.forecastSources {
					if strings.EqualFold(source.Name(), provider) {
						// This is an on-demand fetch for this provider
						ctx := r.Context()
						forecast, err := source.FetchForecast(ctx, location, days)
						if err != nil {
							w.WriteHeader(http.StatusInternalServerError)
							json.NewEncoder(w).Encode(map[string]string{
								"error": fmt.Sprintf("Failed to fetch forecast: %v", err),
							})
							return
						}

						// Store the forecast for future use
						s.forecastStore.UpdateForecast(forecast)

						// Return the forecast
						response := map[string]interface{}{
							"location":  location,
							"provider":  provider,
							"data":      forecast,
							"timestamp": time.Now(),
							"note":      "On-demand forecast fetch",
						}
						w.WriteHeader(http.StatusOK)
						json.NewEncoder(w).Encode(response)
						return
					}
				}
			}

			// If we get here, we couldn't find or fetch the forecast
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("No forecast data found for location '%s' from provider '%s'", location, provider),
			})
			return
		}

		response := map[string]interface{}{
			"location":  location,
			"provider":  provider,
			"data":      forecast,
			"timestamp": time.Now(),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Otherwise return all providers' forecasts for this location
	forecasts, exists := s.forecastStore.GetForecastByLocation(location)
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("No forecast data found for location: %s", location),
		})
		return
	}

	response := map[string]interface{}{
		"location":  location,
		"forecasts": forecasts,
		"timestamp": time.Now(),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleHealthCheck provides a simple health check endpoint
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
