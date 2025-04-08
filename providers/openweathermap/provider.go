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

// OpenWeatherMapSource is an implementation of the DataSource interface for OpenWeatherMap
type OpenWeatherMapSource struct {
	apiKey string
	client *http.Client
}

// Ensure OpenWeatherMapSource implements datasource.DataSource
var _ datasource.DataSource = (*OpenWeatherMapSource)(nil)

// NewOpenWeatherMapSource creates a new OpenWeatherMap data source
func NewOpenWeatherMapSource(apiKey string) *OpenWeatherMapSource {
	return &OpenWeatherMapSource{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Name returns the name of this data source
func (o *OpenWeatherMapSource) Name() string {
	return "OpenWeatherMap"
}

// OpenWeatherMapResponse represents the API response structure
type OpenWeatherMapResponse struct {
	Main struct {
		Temp     float64 `json:"temp"`
		Pressure float64 `json:"pressure"`
		Humidity float64 `json:"humidity"`
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
	Dt   int64  `json:"dt"`
	Sys  struct {
		Sunrise int64 `json:"sunrise"`
		Sunset  int64 `json:"sunset"`
	} `json:"sys"`
}

// FetchWeatherData fetches weather data from the OpenWeatherMap API
func (o *OpenWeatherMapSource) FetchWeatherData(ctx context.Context, location string) (models.WeatherData, error) {
	// Use the current weather endpoint with the q parameter for location
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s&units=metric",
		url.QueryEscape(location), // Properly escape the location parameter
		o.apiKey)

	fmt.Printf("Making API request to: %s\n", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.WeatherData{}, fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	var owmResp OpenWeatherMapResponse
	rawData, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(rawData, &owmResp); err != nil {
		return models.WeatherData{}, fmt.Errorf("failed to parse API response: %w", err)
	}

	// Build the WeatherData struct from the API response
	data := models.WeatherData{
		Provider:    o.Name(),
		Location:    location,
		Timestamp:   time.Unix(owmResp.Dt, 0),
		Temperature: owmResp.Main.Temp,
		Humidity:    owmResp.Main.Humidity,
		WindSpeed:   owmResp.Wind.Speed,
		Pressure:    owmResp.Main.Pressure,
		Description: "",
		Icon:        "",
		WindDeg:     owmResp.Wind.Deg,
		Sunrise:     time.Unix(owmResp.Sys.Sunrise, 0),
		Sunset:      time.Unix(owmResp.Sys.Sunset, 0),
	}

	// Add the weather description and icon if available
	if len(owmResp.Weather) > 0 {
		data.Description = owmResp.Weather[0].Description
		data.Icon = owmResp.Weather[0].Icon
	}

	return data, nil
}
