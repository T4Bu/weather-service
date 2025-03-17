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

// WeatherAPIProvider implements both WeatherProvider and ForecastSource interfaces
type WeatherAPIProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewWeatherAPIProvider creates a new WeatherAPI provider
func NewWeatherAPIProvider(apiKey string) *WeatherAPIProvider {
	return &WeatherAPIProvider{
		apiKey:  apiKey,
		baseURL: "https://api.weatherapi.com/v1",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *WeatherAPIProvider) Name() string {
	return "WeatherAPI"
}

// GetWeather fetches current weather for a location
func (p *WeatherAPIProvider) GetWeather(ctx context.Context, location string) (models.WeatherData, error) {
	// Build URL
	endpoint := fmt.Sprintf("%s/current.json", p.baseURL)
	params := url.Values{}
	params.Add("q", location)
	params.Add("key", p.apiKey)

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
		Location struct {
			Name    string `json:"name"`
			Country string `json:"country"`
		} `json:"location"`
		Current struct {
			TempC      float64 `json:"temp_c"`
			Humidity   int     `json:"humidity"`
			WindKph    float64 `json:"wind_kph"`
			WindDegree int     `json:"wind_degree"`
			PressureMb float64 `json:"pressure_mb"`
			Condition  struct {
				Text string `json:"text"`
				Icon string `json:"icon"`
			} `json:"condition"`
			LastUpdated string `json:"last_updated"`
		} `json:"current"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Create weather data
	return models.WeatherData{
		Provider:    p.Name(),
		Location:    fmt.Sprintf("%s,%s", response.Location.Name, response.Location.Country),
		Temperature: response.Current.TempC,
		Humidity:    float64(response.Current.Humidity),
		WindSpeed:   response.Current.WindKph / 3.6, // Convert to m/s
		Pressure:    response.Current.PressureMb,
		Description: response.Current.Condition.Text,
		Icon:        response.Current.Condition.Icon,
		Timestamp:   time.Now(),
	}, nil
}

// FetchForecast fetches forecast for a location for the specified number of days
func (p *WeatherAPIProvider) FetchForecast(ctx context.Context, location string, days int) (models.ForecastData, error) {
	// Build URL
	endpoint := fmt.Sprintf("%s/forecast.json", p.baseURL)
	params := url.Values{}
	params.Add("q", location)
	params.Add("key", p.apiKey)
	params.Add("days", fmt.Sprintf("%d", days))

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
		Location struct {
			Name    string `json:"name"`
			Country string `json:"country"`
		} `json:"location"`
		Forecast struct {
			ForecastDay []struct {
				Date      string `json:"date"`
				DateEpoch int64  `json:"date_epoch"`
				Day       struct {
					AvgTempC      float64 `json:"avgtemp_c"`
					MaxTempC      float64 `json:"maxtemp_c"`
					MinTempC      float64 `json:"mintemp_c"`
					AvgHumidity   float64 `json:"avghumidity"`
					MaxWindKph    float64 `json:"maxwind_kph"`
					TotalPrecipMm float64 `json:"totalprecip_mm"`
					Condition     struct {
						Text string `json:"text"`
						Icon string `json:"icon"`
					} `json:"condition"`
				} `json:"day"`
				Hour []struct {
					TimeEpoch  int64   `json:"time_epoch"`
					TempC      float64 `json:"temp_c"`
					Humidity   int     `json:"humidity"`
					WindKph    float64 `json:"wind_kph"`
					WindDegree int     `json:"wind_degree"`
					PressureMb float64 `json:"pressure_mb"`
					Condition  struct {
						Text string `json:"text"`
						Icon string `json:"icon"`
					} `json:"condition"`
				} `json:"hour"`
			} `json:"forecastday"`
		} `json:"forecast"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return models.ForecastData{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Process forecast data
	forecast := models.ForecastData{
		Provider:  p.Name(),
		Location:  fmt.Sprintf("%s,%s", response.Location.Name, response.Location.Country),
		Forecasts: []models.Forecast{},
		Updated:   time.Now(),
	}

	// Process hourly forecasts for each day
	for _, day := range response.Forecast.ForecastDay {
		for _, hour := range day.Hour {
			// Convert timestamp
			timestamp := time.Unix(hour.TimeEpoch, 0)

			forecast.Forecasts = append(forecast.Forecasts, models.Forecast{
				Temperature: hour.TempC,
				Humidity:    float64(hour.Humidity),
				WindSpeed:   hour.WindKph / 3.6, // Convert to m/s
				WindDeg:     hour.WindDegree,
				Pressure:    hour.PressureMb,
				Description: hour.Condition.Text,
				Icon:        hour.Condition.Icon,
				Timestamp:   timestamp,
			})
		}
	}

	return forecast, nil
}
