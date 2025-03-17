package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"weather-service/datasource"
	"weather-service/models"
)

// DataCollector manages the collection of weather data from multiple sources
type DataCollector struct {
	sources      []datasource.DataSource
	outputChan   chan models.WeatherData
	errorChan    chan error
	locations    []string
	fetchTimeout time.Duration
}

// NewDataCollector creates a new data collector with the provided sources
func NewDataCollector(sources []datasource.DataSource, locations []string) *DataCollector {
	return &DataCollector{
		sources:      sources,
		outputChan:   make(chan models.WeatherData, 100), // Buffer size can be configured
		errorChan:    make(chan error, 100),              // Buffer for errors
		locations:    locations,
		fetchTimeout: 10 * time.Second, // Default timeout
	}
}

// SetFetchTimeout changes the timeout for API requests
func (dc *DataCollector) SetFetchTimeout(timeout time.Duration) {
	dc.fetchTimeout = timeout
}

// OutputChannel returns the channel that emits collected weather data
func (dc *DataCollector) OutputChannel() <-chan models.WeatherData {
	return dc.outputChan
}

// ErrorChannel returns the channel that emits errors
func (dc *DataCollector) ErrorChannel() <-chan error {
	return dc.errorChan
}

// Start begins collecting data from all sources for all locations
// The returned function can be called to stop collection
func (dc *DataCollector) Start(ctx context.Context) func() {
	// Create a new context that we can cancel
	collectionCtx, cancelCollection := context.WithCancel(ctx)

	var wg sync.WaitGroup

	// Start collection for each source and location combination
	for _, source := range dc.sources {
		for _, location := range dc.locations {
			wg.Add(1)
			go dc.collectFromSource(collectionCtx, &wg, source, location)
		}
	}

	// Start a goroutine that will close channels when all collectors are done
	go func() {
		wg.Wait()
		close(dc.outputChan)
		close(dc.errorChan)
	}()

	// Return a function that will stop all collection when called
	return func() {
		cancelCollection()
		// Wait for everything to clean up
		wg.Wait()
	}
}

// collectFromSource continuously collects data from a single source for a location
func (dc *DataCollector) collectFromSource(ctx context.Context, wg *sync.WaitGroup, source datasource.DataSource, location string) {
	defer wg.Done()

	ticker := time.NewTicker(15 * time.Minute) // Fetch every 15 minutes by default
	defer ticker.Stop()

	// Do an initial fetch immediately
	dc.fetchOnce(ctx, source, location)

	// Then fetch on the ticker schedule
	for {
		select {
		case <-ticker.C:
			dc.fetchOnce(ctx, source, location)
		case <-ctx.Done():
			return
		}
	}
}

// fetchOnce performs a single fetch from a data source
func (dc *DataCollector) fetchOnce(ctx context.Context, source datasource.DataSource, location string) {
	// Create a context with timeout for this specific request
	fetchCtx, cancel := context.WithTimeout(ctx, dc.fetchTimeout)
	defer cancel()

	// Fetch the data
	data, err := source.FetchWeatherData(fetchCtx, location)
	if err != nil {
		select {
		case dc.errorChan <- fmt.Errorf("error fetching from %s for %s: %w", source.Name(), location, err):
		default:
			// If error channel is full, log or handle differently if needed
		}
		return
	}

	// Send the data to the output channel
	select {
	case dc.outputChan <- data:
	case <-ctx.Done():
		return
	}
}
