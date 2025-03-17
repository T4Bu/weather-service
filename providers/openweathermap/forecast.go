package openweathermap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"weather-service/datasource"
	"weather-service/models"
)

// OpenWeatherMapForecastSource provides forecasts from OpenWeatherMap
type OpenWeatherMapForecastSource struct {
	apiKey string
	client *http.Client
}

// Ensure OpenWeatherMapForecastSource implements ForecastSource
var _ datasource.ForecastSource = (*OpenWeatherMapForecastSource)(nil)

// NewOpenWeatherMapForecastSource creates a new forecast source
func NewOpenWeatherMapForecastSource(apiKey string) *OpenWeatherMapForecastSource {
	return &OpenWeatherMapForecastSource{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the provider name
func (o *OpenWeatherMapForecastSource) Name() string {
	return "OpenWeatherMap"
}

// OpenWeatherMapForecastResponse represents the API response structure
type OpenWeatherMapForecastResponse struct {
	City struct {
		Name string `json:"name"`
	} `json:"city"`
	List []struct {
		Dt   int64 `json:"dt"` // Timestamp
		Main struct {
			Temp     float64 `json:"temp"`
			Pressure float64 `json:"pressure"`
			Humidity float64 `json:"humidity"`
		} `json:"main"`
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
		Wind struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
		} `json:"wind"`
		Clouds struct {
			All int `json:"all"` // Cloudiness percentage
		} `json:"clouds"`
		Pop float64 `json:"pop"` // Probability of precipitation
	} `json:"list"`
}

// FetchForecast gets forecast data from OpenWeatherMap
func (o *OpenWeatherMapForecastSource) FetchForecast(ctx context.Context, location string, days int) (models.ForecastData, error) {
	// OpenWeatherMap's free tier forecast API provides 5-day forecasts at 3-hour intervals
	// We'll limit to the requested number of days

	// Build the URL
	apiURL := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?q=%s&appid=%s&units=metric",
		url.QueryEscape(location), o.apiKey)

	fmt.Printf("Making OpenWeatherMap forecast request to: %s\n", apiURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return models.ForecastData{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return models.ForecastData{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.ForecastData{}, fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	// Read the response body
	rawData, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.ForecastData{}, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the response
	var forecastResp OpenWeatherMapForecastResponse
	if err := json.Unmarshal(rawData, &forecastResp); err != nil {
		return models.ForecastData{}, fmt.Errorf("failed to parse API response: %w", err)
	}

	// Create the forecast data
	forecastData := models.ForecastData{
		Provider:  o.Name(),
		Location:  location,
		Forecasts: []models.Forecast{},
		Updated:   time.Now(),
	}

	// Calculate the maximum forecast time based on requested days
	maxForecastTime := time.Now().AddDate(0, 0, days)

	// Convert the forecast entries
	for _, item := range forecastResp.List {
		forecastTime := time.Unix(item.Dt, 0)

		// Skip if beyond requested days
		if forecastTime.After(maxForecastTime) {
			continue
		}

		// Add weather condition if available
		description := ""
		if len(item.Weather) > 0 {
			description = item.Weather[0].Description
		}

		// Create a forecast entry
		forecast := models.Forecast{
			Temperature: item.Main.Temp,
			Humidity:    item.Main.Humidity,
			WindSpeed:   item.Wind.Speed,
			WindDeg:     item.Wind.Deg,
			Pressure:    item.Main.Pressure,
			Description: description,
			Icon:        "", // OpenWeatherMap doesn't provide icon in this response
			Timestamp:   forecastTime,
		}

		forecastData.Forecasts = append(forecastData.Forecasts, forecast)
	}

	return forecastData, nil
}
