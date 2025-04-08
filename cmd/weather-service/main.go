package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"weather-service/api"
	"weather-service/datasource"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Parse command line arguments
	port := flag.Int("port", 8080, "Port to run the server on")
	updateInterval := flag.Duration("update", 5*time.Minute, "Weather data update interval")
	configFile := flag.String("config", "config.json", "Path to configuration file")
	enableRateLimiting := flag.Bool("rate-limit", true, "Enable API rate limiting")
	flag.Parse()

	// Load configuration
	config, err := datasource.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create the providers based on configuration
	var providers []datasource.WeatherProvider
	var forecastSources []datasource.ForecastSource

	if config.OpenWeatherMap.Enabled {
		if config.OpenWeatherMap.APIKey == "" {
			log.Fatal("OpenWeatherMap is enabled but no API key provided")
		}
		log.Printf("Using OpenWeatherMap API key: %s", config.OpenWeatherMap.APIKey)
		owmProvider := datasource.NewOpenWeatherMapProvider(config.OpenWeatherMap.APIKey)

		// Apply rate limiting if enabled
		if *enableRateLimiting {
			// OpenWeatherMap free tier allows 60 calls/minute = 1 call per second
			// Allow bursts of up to 5 requests
			rateLimitedProvider := datasource.NewRateLimitedProvider(owmProvider, 1.0, 1.0, 5)
			providers = append(providers, rateLimitedProvider)
			forecastSources = append(forecastSources, rateLimitedProvider)
			log.Println("Applied rate limiting to OpenWeatherMap provider")
		} else {
			providers = append(providers, owmProvider)
			forecastSources = append(forecastSources, owmProvider)
		}
	}

	if config.WeatherAPI.Enabled {
		if config.WeatherAPI.APIKey == "" {
			log.Fatal("WeatherAPI is enabled but no API key provided")
		}
		log.Printf("Using WeatherAPI API key: %s", config.WeatherAPI.APIKey)
		wapiProvider := datasource.NewWeatherAPIProvider(config.WeatherAPI.APIKey)

		// Apply rate limiting if enabled
		if *enableRateLimiting {
			// WeatherAPI free tier allows ~23 calls/minute = 0.4 calls per second
			// Allow bursts of up to 3 requests
			rateLimitedProvider := datasource.NewRateLimitedProvider(wapiProvider, 0.4, 0.4, 3)
			providers = append(providers, rateLimitedProvider)
			forecastSources = append(forecastSources, rateLimitedProvider)
			log.Println("Applied rate limiting to WeatherAPI provider")
		} else {
			providers = append(providers, wapiProvider)
			forecastSources = append(forecastSources, wapiProvider)
		}
	}

	if len(providers) == 0 {
		log.Fatal("No weather providers enabled in configuration")
	}

	// Create in-memory stores for weather and forecast data
	weatherStore := api.NewWeatherStore()
	forecastStore := api.NewForecastStore()

	// Create API server
	server := api.NewServer(weatherStore, forecastStore, *port)
	server.RegisterForecastSources(forecastSources)

	// Set up channels for graceful shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	updateChan := make(chan struct{})

	// Start data updater in a goroutine
	go func() {
		ticker := time.NewTicker(*updateInterval)
		defer ticker.Stop()

		// Update weather and forecast data immediately on startup
		updateData(providers, forecastSources, weatherStore, forecastStore, config)

		for {
			select {
			case <-ticker.C:
				updateData(providers, forecastSources, weatherStore, forecastStore, config)
			case <-updateChan:
				return
			}
		}
	}()

	// Start the API server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-shutdownChan
	fmt.Printf("Shutting down due to %s signal\n", sig)

	// Notify updater to stop
	close(updateChan)

	// Periodically clean up old forecasts (every 24 hours)
	forecastPruneAge := 48 * time.Hour // Remove forecasts older than 2 days
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				forecastStore.PruneOldForecasts(forecastPruneAge)
			case <-updateChan:
				return
			}
		}
	}()

	fmt.Println("Shutdown complete")
}

// updateData fetches the latest weather and forecast data from all providers
func updateData(
	providers []datasource.WeatherProvider,
	forecastSources []datasource.ForecastSource,
	weatherStore *api.WeatherStore,
	forecastStore *api.ForecastStore,
	config *datasource.Config,
) {
	fmt.Println("Updating weather data...")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create wait group for concurrent updates
	var wg sync.WaitGroup

	// Update current weather data
	for _, location := range config.Locations {
		for _, provider := range providers {
			wg.Add(1)
			go func(loc string, prov datasource.WeatherProvider) {
				defer wg.Done()

				// Get current weather
				data, err := prov.GetWeather(ctx, loc)
				if err != nil {
					log.Printf("Error fetching weather for %s from %s: %v", loc, prov.Name(), err)
					return
				}

				// Store the data
				weatherStore.UpdateWeather(data)
				log.Printf("Updated weather data for %s from %s", loc, prov.Name())
			}(location, provider)
		}
	}

	// Update forecast data (3 days by default)
	for _, location := range config.Locations {
		for _, source := range forecastSources {
			wg.Add(1)
			go func(loc string, src datasource.ForecastSource) {
				defer wg.Done()

				// Get forecast data (3 days)
				forecast, err := src.FetchForecast(ctx, loc, 3)
				if err != nil {
					log.Printf("Error fetching forecast for %s from %s: %v", loc, src.Name(), err)
					return
				}

				// Store the forecast data
				forecastStore.UpdateForecast(forecast)
				log.Printf("Updated forecast data for %s from %s", loc, src.Name())
			}(location, source)
		}
	}

	// Wait for all updates to complete
	wg.Wait()
	fmt.Println("Weather and forecast data update complete")
}
