package weatherapi

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

// WeatherAPISource is an implementation of the DataSource interface for WeatherAPI.com
type WeatherAPISource struct {
	apiKey string
	client *http.Client
}

// Ensure WeatherAPISource implements datasource.DataSource
var _ datasource.DataSource = (*WeatherAPISource)(nil)

// NewWeatherAPISource creates a new WeatherAPI data source
func NewWeatherAPISource(apiKey string) *WeatherAPISource {
	return &WeatherAPISource{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Name returns the name of this data source
func (w *WeatherAPISource) Name() string {
	return "WeatherAPI"
}

// WeatherAPIResponse represents the API response structure
type WeatherAPIResponse struct {
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
	Forecast struct {
		Forecastday []struct {
			Astro struct {
				Sunrise string `json:"sunrise"`
				Sunset  string `json:"sunset"`
			} `json:"astro"`
		} `json:"forecastday"`
	} `json:"forecast"`
}

// FetchWeatherData fetches weather data from the WeatherAPI.com API
func (w *WeatherAPISource) FetchWeatherData(ctx context.Context, location string) (models.WeatherData, error) {
	// Use the current weather endpoint
	url := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s",
		w.apiKey,
		url.QueryEscape(location)) // Properly escape the location parameter

	fmt.Printf("Making WeatherAPI request to: %s\n", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.WeatherData{}, fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	var wapiResp WeatherAPIResponse
	rawData, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(rawData, &wapiResp); err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to parse API response: %w", err)
	}

	// Parse the last updated time if available
	lastUpdated := time.Now()
	if wapiResp.Current.LastUpdated != "" {
		if t, err := time.Parse("2006-01-02 15:04", wapiResp.Current.LastUpdated); err == nil {
			lastUpdated = t
		}
	}

	// Parse sunrise and sunset times
	var sunrise, sunset time.Time
	if len(wapiResp.Forecast.Forecastday) > 0 {
		astro := wapiResp.Forecast.Forecastday[0].Astro
		if t, err := time.Parse("hh:mm AM", astro.Sunrise); err == nil {
			sunrise = t
		}
		if t, err := time.Parse("hh:mm AM", astro.Sunset); err == nil {
			sunset = t
		}
	}

	// Format the location with country if available
	formattedLocation := location
	if wapiResp.Location.Name != "" && wapiResp.Location.Country != "" {
		formattedLocation = fmt.Sprintf("%s,%s", wapiResp.Location.Name, wapiResp.Location.Country)
	}

	// Build the WeatherData struct from the API response
	return models.WeatherData{
		Provider:    w.Name(),
		Location:    formattedLocation,
		Timestamp:   lastUpdated,
		Temperature: wapiResp.Current.TempC,
		Humidity:    float64(wapiResp.Current.Humidity),
		WindSpeed:   wapiResp.Current.WindKph / 3.6, // Convert km/h to m/s
		WindDeg:     wapiResp.Current.WindDegree,
		Pressure:    wapiResp.Current.PressureMb,
		Description: wapiResp.Current.Condition.Text,
		Icon:        wapiResp.Current.Condition.Icon,
		Sunrise:     sunrise,
		Sunset:      sunset,
	}, nil
}
