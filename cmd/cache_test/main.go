package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"weather-service/cache"
	"weather-service/datasource"
	"weather-service/providers/openweathermap"
	"weather-service/providers/weatherapi"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("=== Running Cache Test ===")
	fmt.Println("This will demonstrate how caching works with multiple requests")
	fmt.Println("The test will take about 30 seconds to complete...\n")

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}

	// Get API keys from environment variables
	openWeatherMapKey := os.Getenv("OPENWEATHERMAP_API_KEY")
	weatherAPIKey := os.Getenv("WEATHERAPI_KEY")

	// Set a short cache duration for demonstration purposes
	cacheDuration := 15 * time.Second // Shorter for quicker demo

	// Create sources with caching
	var sources []datasource.DataSource

	if openWeatherMapKey != "" {
		openWeatherSource := openweathermap.NewOpenWeatherMapSource(openWeatherMapKey)
		cachedOpenWeatherSource := cache.NewCachedDataSource(openWeatherSource, cacheDuration)
		sources = append(sources, cachedOpenWeatherSource)
		fmt.Println("Added OpenWeatherMap source with 15-second cache")
	}

	if weatherAPIKey != "" {
		weatherAPISource := weatherapi.NewWeatherAPISource(weatherAPIKey)
		cachedWeatherAPISource := cache.NewCachedDataSource(weatherAPISource, cacheDuration)
		sources = append(sources, cachedWeatherAPISource)
		fmt.Println("Added WeatherAPI source with 15-second cache")
	}

	if len(sources) == 0 {
		log.Fatal("No API keys provided")
	}

	ctx := context.Background()
	locations := []string{"London,UK", "New York,US"}

	fmt.Println("\n*** First Request - Should be cache misses ***")
	makeRequests(ctx, sources, locations)

	fmt.Println("\n*** Second Request - Should use cached data ***")
	makeRequests(ctx, sources, locations)

	fmt.Println("\n*** Third Request - Still using cached data ***")
	makeRequests(ctx, sources, locations)

	fmt.Println("\nWaiting for cache to expire (15 seconds)...")
	time.Sleep(cacheDuration + 1*time.Second)

	fmt.Println("\n*** After Expiry - Should be cache misses again ***")
	makeRequests(ctx, sources, locations)

	// Check cache stats - each source should have 3 hits and 2 misses per location
	for _, source := range sources {
		if cachedSource, ok := source.(*cache.CachedDataSource); ok {
			hits, misses := cachedSource.CacheStats()
			fmt.Printf("\nStats for %s: %d cache hits, %d cache misses\n",
				cachedSource.Name(), hits, misses)
		}
	}

	fmt.Println("\n=== Cache Test Complete ===")
}

func makeRequests(ctx context.Context, sources []datasource.DataSource, locations []string) {
	for _, source := range sources {
		for _, location := range locations {
			data, err := source.FetchWeatherData(ctx, location)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Printf("Got data from %s for %s: %.1f°C\n",
				data.Provider, location, data.Temperature)
		}
	}
}
