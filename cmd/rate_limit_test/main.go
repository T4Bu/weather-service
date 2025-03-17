package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"weather-service/datasource"
	"weather-service/models"
)

// MockWeatherProvider is a simple mock that simulates latency and counts calls
type MockWeatherProvider struct {
	callCount   int
	mutex       sync.Mutex
	latency     time.Duration
	shouldFail  bool
	failAfter   int
	maxRequests int
}

func NewMockWeatherProvider(latency time.Duration, shouldFail bool, failAfter int) *MockWeatherProvider {
	return &MockWeatherProvider{
		latency:    latency,
		shouldFail: shouldFail,
		failAfter:  failAfter,
	}
}

func (m *MockWeatherProvider) GetWeather(ctx context.Context, location string) (models.WeatherData, error) {
	m.mutex.Lock()
	m.callCount++
	currentCount := m.callCount
	m.mutex.Unlock()

	// Log request time
	now := time.Now()
	fmt.Printf("%s - Processing request #%d for %s\n", now.Format("15:04:05.000"), currentCount, location)

	// Simulate work/latency
	select {
	case <-time.After(m.latency):
		// Continue processing
	case <-ctx.Done():
		return models.WeatherData{}, ctx.Err()
	}

	// Check if we should fail after a certain number of requests
	if m.shouldFail && currentCount > m.failAfter {
		return models.WeatherData{}, fmt.Errorf("service unavailable (too many requests)")
	}

	return models.WeatherData{
		Location:    location,
		Provider:    m.Name(),
		Temperature: 22.5,
		Humidity:    60,
		WindSpeed:   5.5,
		Description: "Mocked weather data",
		Timestamp:   time.Now(),
	}, nil
}

func (m *MockWeatherProvider) Name() string {
	return "MockProvider"
}

func (m *MockWeatherProvider) GetCallCount() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.callCount
}

func main() {
	// Parse command-line flags
	requestsPerSecond := flag.Float64("rps", 1.0, "Rate limit in requests per second")
	burstSize := flag.Int("burst", 3, "Maximum burst size")
	totalRequests := flag.Int("requests", 10, "Total number of requests to make")
	concurrentRequests := flag.Int("concurrent", 5, "Number of concurrent requests")
	flag.Parse()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a mock provider with 200ms response time
	mockProvider := NewMockWeatherProvider(200*time.Millisecond, false, 0)

	// Wrap with rate limiter
	rateLimitedProvider := datasource.NewRateLimitedWeatherProvider(mockProvider, *requestsPerSecond, *burstSize)

	fmt.Printf("Testing rate limiter with:\n")
	fmt.Printf("- Rate limit: %.2f requests/second\n", *requestsPerSecond)
	fmt.Printf("- Burst size: %d\n", *burstSize)
	fmt.Printf("- Total requests: %d\n", *totalRequests)
	fmt.Printf("- Concurrent workers: %d\n", *concurrentRequests)
	fmt.Println("Starting test...")

	// Record start time
	startTime := time.Now()

	// Create wait group for concurrent requests
	var wg sync.WaitGroup

	// Launch concurrent goroutines
	for i := 0; i < *concurrentRequests; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Calculate how many requests this worker should make
			requestsPerWorker := *totalRequests / *concurrentRequests
			if workerID < *totalRequests%*concurrentRequests {
				requestsPerWorker++
			}

			// Make requests
			for j := 0; j < requestsPerWorker; j++ {
				location := fmt.Sprintf("TestLocation-%d-%d", workerID, j)
				before := time.Now()
				_, err := rateLimitedProvider.GetWeather(ctx, location)
				elapsed := time.Since(before)

				if err != nil {
					log.Printf("Worker %d - Request %d failed: %v", workerID, j, err)
				} else {
					log.Printf("Worker %d - Request %d completed in %v", workerID, j, elapsed)
				}

				// Small sleep to prevent tight loop
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Calculate total time
	totalTime := time.Since(startTime)
	actualRPS := float64(*totalRequests) / totalTime.Seconds()

	fmt.Println("\nTest completed!")
	fmt.Printf("Total time: %.2f seconds\n", totalTime.Seconds())
	fmt.Printf("Actual requests per second: %.2f\n", actualRPS)
	fmt.Printf("Total requests processed: %d\n", mockProvider.GetCallCount())

	expectedMinTime := float64(*totalRequests-*burstSize) / *requestsPerSecond
	if expectedMinTime < 0 {
		expectedMinTime = 0
	}

	fmt.Printf("Expected minimum time (theoretical): %.2f seconds\n", expectedMinTime)

	if actualRPS > *requestsPerSecond*1.5 && *totalRequests > *burstSize {
		fmt.Println("\n⚠️ WARNING: Actual RPS significantly higher than configured rate limit!")
		fmt.Println("Rate limiting may not be working as expected.")
	} else {
		fmt.Println("\n✅ Rate limiting appears to be working correctly.")
	}
}
