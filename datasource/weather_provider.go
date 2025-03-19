package datasource

import (
	"context"
	"encoding/json"
	"os"

	"weather-service/models"
)

// WeatherProvider is an interface for services that can fetch current weather data
type WeatherProvider interface {
	// GetWeather fetches current weather for a location
	GetWeather(ctx context.Context, location string) (models.WeatherData, error)

	// Name returns the provider's name
	Name() string
}

// ForecastSource is an interface for services that can fetch weather forecasts
type ForecastSource interface {
	// FetchForecast fetches forecast for a location for the specified number of days
	FetchForecast(ctx context.Context, location string, days int) (models.ForecastData, error)

	// Name returns the source's name
	Name() string
}

// Config represents the application configuration
type Config struct {
	// API provider configurations
	OpenWeatherMap struct {
		Enabled bool   `json:"enabled"`
		APIKey  string `json:"apiKey"`
	} `json:"openWeatherMap"`

	WeatherAPI struct {
		Enabled bool   `json:"enabled"`
		APIKey  string `json:"apiKey"`
	} `json:"weatherAPI"`

	// List of locations to monitor
	Locations []string `json:"locations"`
}

// LoadConfig loads configuration from a JSON file and environment variables
func LoadConfig(filename string) (*Config, error) {
	// Load base configuration from JSON file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	// Override API keys with environment variables if they exist
	if apiKey := os.Getenv("OPENWEATHERMAP_API_KEY"); apiKey != "" {
		config.OpenWeatherMap.APIKey = apiKey
	}
	if apiKey := os.Getenv("WEATHERAPI_KEY"); apiKey != "" {
		config.WeatherAPI.APIKey = apiKey
	}

	return &config, nil
}

// DefaultConfig creates a default configuration
func DefaultConfig() *Config {
	config := &Config{}
	config.OpenWeatherMap.Enabled = false
	config.WeatherAPI.Enabled = false
	config.Locations = []string{"London,UK", "New York,US", "Tokyo,JP"}
	return config
}
