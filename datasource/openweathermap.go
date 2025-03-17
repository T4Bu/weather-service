package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"weather-service/models"
)

// OpenWeatherMapProvider implements both WeatherProvider and ForecastSource interfaces
type OpenWeatherMapProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOpenWeatherMapProvider creates a new OpenWeatherMap provider
func NewOpenWeatherMapProvider(apiKey string) *OpenWeatherMapProvider {
	return &OpenWeatherMapProvider{
		apiKey:  apiKey,
		baseURL: "https://api.openweathermap.org/data/2.5",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *OpenWeatherMapProvider) Name() string {
	return "OpenWeatherMap"
}

// GetWeather fetches current weather for a location
func (p *OpenWeatherMapProvider) GetWeather(ctx context.Context, location string) (models.WeatherData, error) {
	// Build URL
	endpoint := fmt.Sprintf("%s/weather", p.baseURL)
	params := url.Values{}
	params.Add("q", location)
	params.Add("appid", p.apiKey)
	params.Add("units", "metric") // Use metric units

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error status code
	if resp.StatusCode != http.StatusOK {
		return models.WeatherData{}, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		Main struct {
			Temp     float64 `json:"temp"`
			Humidity int     `json:"humidity"`
			Pressure int     `json:"pressure"`
		} `json:"main"`
		Wind struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
		} `json:"wind"`
		Weather []struct {
			Description string `json:"description"`
			Icon        string `json:"icon"`
		} `json:"weather"`
		Name string `json:"name"`
		Sys  struct {
			Country string `json:"country"`
		} `json:"sys"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract weather description and icon if available
	description := ""
	icon := ""
	if len(response.Weather) > 0 {
		description = response.Weather[0].Description
		icon = response.Weather[0].Icon
	}

	// Format location
	formattedLocation := response.Name
	if response.Sys.Country != "" {
		formattedLocation = fmt.Sprintf("%s,%s", response.Name, response.Sys.Country)
	}

	// Create weather data
	return models.WeatherData{
		Provider:    p.Name(),
		Location:    formattedLocation,
		Temperature: response.Main.Temp,
		Humidity:    float64(response.Main.Humidity),
		WindSpeed:   response.Wind.Speed,
		WindDeg:     response.Wind.Deg,
		Pressure:    float64(response.Main.Pressure),
		Description: description,
		Icon:        icon,
		Timestamp:   time.Now(),
	}, nil
}

// FetchForecast fetches forecast for a location for the specified number of days
func (p *OpenWeatherMapProvider) FetchForecast(ctx context.Context, location string, days int) (models.ForecastData, error) {
	// OpenWeatherMap's 5-day forecast endpoint returns data in 3-hour steps
	endpoint := fmt.Sprintf("%s/forecast", p.baseURL)
	params := url.Values{}
	params.Add("q", location)
	params.Add("appid", p.apiKey)
	params.Add("units", "metric") // Use metric units

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return models.ForecastData{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return models.ForecastData{}, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.ForecastData{}, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error status code
	if resp.StatusCode != http.StatusOK {
		return models.ForecastData{}, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response struct {
		City struct {
			Name    string `json:"name"`
			Country string `json:"country"`
		} `json:"city"`
		List []struct {
			Main struct {
				Temp     float64 `json:"temp"`
				Humidity int     `json:"humidity"`
				Pressure int     `json:"pressure"`
			} `json:"main"`
			Wind struct {
				Speed float64 `json:"speed"`
				Deg   int     `json:"deg"`
			} `json:"wind"`
			Weather []struct {
				Description string `json:"description"`
				Icon        string `json:"icon"`
			} `json:"weather"`
			Dt    int64  `json:"dt"`
			DtTxt string `json:"dt_txt"`
		} `json:"list"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return models.ForecastData{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Process forecast data
	forecast := models.ForecastData{
		Provider:  p.Name(),
		Location:  fmt.Sprintf("%s,%s", response.City.Name, response.City.Country),
		Forecasts: []models.Forecast{},
		Updated:   time.Now(),
	}

	// Number of entries to include (8 entries per day, as they come in 3-hour intervals)
	maxEntries := days * 8
	if maxEntries > len(response.List) {
		maxEntries = len(response.List)
	}

	// Convert response to our model
	for i := 0; i < maxEntries; i++ {
		item := response.List[i]

		// Get weather description and icon if available
		description := ""
		icon := ""
		if len(item.Weather) > 0 {
			description = item.Weather[0].Description
			icon = item.Weather[0].Icon
		}

		// Convert timestamp
		timestamp := time.Unix(item.Dt, 0)

		forecast.Forecasts = append(forecast.Forecasts, models.Forecast{
			Temperature: item.Main.Temp,
			Humidity:    float64(item.Main.Humidity),
			WindSpeed:   item.Wind.Speed,
			WindDeg:     item.Wind.Deg,
			Pressure:    float64(item.Main.Pressure),
			Description: description,
			Icon:        icon,
			Timestamp:   timestamp,
		})
	}

	return forecast, nil
}
