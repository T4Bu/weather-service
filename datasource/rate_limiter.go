package datasource

import (
	"context"
	"fmt"

	"weather-service/models"

	"golang.org/x/time/rate"
)

// RateLimitedWeatherProvider wraps a WeatherProvider with rate limiting
type RateLimitedWeatherProvider struct {
	provider WeatherProvider
	limiter  *rate.Limiter
	name     string
}

// NewRateLimitedWeatherProvider creates a new rate limited weather provider
// rps is the maximum requests per second allowed (can be fractional for less than 1 request per second)
// burst is the maximum burst size allowed
func NewRateLimitedWeatherProvider(provider WeatherProvider, rps float64, burst int) *RateLimitedWeatherProvider {
	return &RateLimitedWeatherProvider{
		provider: provider,
		limiter:  rate.NewLimiter(rate.Limit(rps), burst),
		name:     fmt.Sprintf("%s [Rate Limited]", provider.Name()),
	}
}

// GetWeather fetches weather data, respecting rate limits
func (r *RateLimitedWeatherProvider) GetWeather(ctx context.Context, location string) (models.WeatherData, error) {
	// Wait for rate limiter permission or context cancellation
	if err := r.limiter.Wait(ctx); err != nil {
		return models.WeatherData{}, fmt.Errorf("rate limit wait canceled: %w", err)
	}

	// Forward to the underlying provider
	return r.provider.GetWeather(ctx, location)
}

// Name returns the provider name
func (r *RateLimitedWeatherProvider) Name() string {
	return r.name
}

// RateLimitedForecastSource wraps a ForecastSource with rate limiting
type RateLimitedForecastSource struct {
	source  ForecastSource
	limiter *rate.Limiter
	name    string
}

// NewRateLimitedForecastSource creates a new rate limited forecast source
// rps is the maximum requests per second allowed
// burst is the maximum burst size allowed
func NewRateLimitedForecastSource(source ForecastSource, rps float64, burst int) *RateLimitedForecastSource {
	return &RateLimitedForecastSource{
		source:  source,
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
		name:    fmt.Sprintf("%s [Rate Limited]", source.Name()),
	}
}

// FetchForecast fetches forecast data, respecting rate limits
func (r *RateLimitedForecastSource) FetchForecast(ctx context.Context, location string, days int) (models.ForecastData, error) {
	// Wait for rate limiter permission or context cancellation
	if err := r.limiter.Wait(ctx); err != nil {
		return models.ForecastData{}, fmt.Errorf("rate limit wait canceled: %w", err)
	}

	// Forward to the underlying source
	return r.source.FetchForecast(ctx, location, days)
}

// Name returns the source name
func (r *RateLimitedForecastSource) Name() string {
	return r.name
}

// RateLimitedProvider combines both interfaces for providers that implement both
type RateLimitedProvider struct {
	provider        WeatherProvider
	forecastSrc     ForecastSource
	weatherLimiter  *rate.Limiter
	forecastLimiter *rate.Limiter
	name            string
}

// NewRateLimitedProvider creates a provider that implements both interfaces with rate limiting
// weatherRPS and forecastRPS are the maximum requests per second for weather and forecast APIs
func NewRateLimitedProvider(provider interface{}, weatherRPS, forecastRPS float64, burst int) *RateLimitedProvider {
	name := "Unknown"

	// Type assertions to get the name
	if wp, ok := provider.(WeatherProvider); ok {
		name = wp.Name()
	} else if fs, ok := provider.(ForecastSource); ok {
		name = fs.Name()
	}

	return &RateLimitedProvider{
		provider:        provider.(WeatherProvider),
		forecastSrc:     provider.(ForecastSource),
		weatherLimiter:  rate.NewLimiter(rate.Limit(weatherRPS), burst),
		forecastLimiter: rate.NewLimiter(rate.Limit(forecastRPS), burst),
		name:            fmt.Sprintf("%s [Rate Limited]", name),
	}
}

// GetWeather implements WeatherProvider interface with rate limiting
func (r *RateLimitedProvider) GetWeather(ctx context.Context, location string) (models.WeatherData, error) {
	if err := r.weatherLimiter.Wait(ctx); err != nil {
		return models.WeatherData{}, fmt.Errorf("rate limit wait canceled: %w", err)
	}
	return r.provider.GetWeather(ctx, location)
}

// FetchForecast implements ForecastSource interface with rate limiting
func (r *RateLimitedProvider) FetchForecast(ctx context.Context, location string, days int) (models.ForecastData, error) {
	if err := r.forecastLimiter.Wait(ctx); err != nil {
		return models.ForecastData{}, fmt.Errorf("rate limit wait canceled: %w", err)
	}
	return r.forecastSrc.FetchForecast(ctx, location, days)
}

// Name returns the provider name
func (r *RateLimitedProvider) Name() string {
	return r.name
}

// Verify that our rate limited types implement the required interfaces
var (
	_ WeatherProvider = (*RateLimitedWeatherProvider)(nil)
	_ ForecastSource  = (*RateLimitedForecastSource)(nil)
	_ WeatherProvider = (*RateLimitedProvider)(nil)
	_ ForecastSource  = (*RateLimitedProvider)(nil)
)
