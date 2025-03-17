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

// WeatherAPIForecastSource provides forecasts from WeatherAPI.com
type WeatherAPIForecastSource struct {
	apiKey string
	client *http.Client
}

// Ensure WeatherAPIForecastSource implements ForecastSource
var _ datasource.ForecastSource = (*WeatherAPIForecastSource)(nil)

// NewWeatherAPIForecastSource creates a new forecast source
func NewWeatherAPIForecastSource(apiKey string) *WeatherAPIForecastSource {
	return &WeatherAPIForecastSource{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the provider name
func (w *WeatherAPIForecastSource) Name() string {
	return "WeatherAPI"
}

// WeatherAPIForecastResponse represents the API response structure
type WeatherAPIForecastResponse struct {
	Location struct {
		Name string `json:"name"`
	} `json:"location"`
	Forecast struct {
		ForecastDay []struct {
			Date      string `json:"date"`
			DateEpoch int64  `json:"date_epoch"`
			Day       struct {
				MaxTempC          float64 `json:"maxtemp_c"`
				MinTempC          float64 `json:"mintemp_c"`
				AvgTempC          float64 `json:"avgtemp_c"`
				MaxWindKph        float64 `json:"maxwind_kph"`
				TotalPrecipMm     float64 `json:"totalprecip_mm"`
				AvgHumidity       float64 `json:"avghumidity"`
				DailyChanceOfRain int     `json:"daily_chance_of_rain"`
				Condition         struct {
					Text string `json:"text"`
				} `json:"condition"`
			} `json:"day"`
			Hour []struct {
				TimeEpoch int64   `json:"time_epoch"`
				Time      string  `json:"time"`
				TempC     float64 `json:"temp_c"`
				Condition struct {
					Text string `json:"text"`
					Icon string `json:"icon"`
				} `json:"condition"`
				WindKph      float64 `json:"wind_kph"`
				WindDegree   int     `json:"wind_degree"`
				PressureMb   float64 `json:"pressure_mb"`
				Humidity     int     `json:"humidity"`
				ChanceOfRain int     `json:"chance_of_rain"`
			} `json:"hour"`
		} `json:"forecastday"`
	} `json:"forecast"`
}

// FetchForecast gets forecast data from WeatherAPI.com
func (w *WeatherAPIForecastSource) FetchForecast(ctx context.Context, location string, days int) (models.ForecastData, error) {
	// WeatherAPI.com can provide up to 14 days of forecast data in the paid tier
	// For the free tier, it's limited to 3 days
	if days > 3 {
		days = 3 // Limit to what the free tier can provide
	}

	// Build the URL
	apiURL := fmt.Sprintf("https://api.weatherapi.com/v1/forecast.json?key=%s&q=%s&days=%d&aqi=no&alerts=no",
		w.apiKey, url.QueryEscape(location), days)

	fmt.Printf("Making WeatherAPI forecast request to: %s\n", apiURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return models.ForecastData{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := w.client.Do(req)
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
	var forecastResp WeatherAPIForecastResponse
	if err := json.Unmarshal(rawData, &forecastResp); err != nil {
		return models.ForecastData{}, fmt.Errorf("failed to parse API response: %w", err)
	}

	// Create the forecast data
	forecastData := models.ForecastData{
		Provider:  w.Name(),
		Location:  location,
		Forecasts: []models.Forecast{},
		Updated:   time.Now(),
	}

	// Process each forecast day
	for _, day := range forecastResp.Forecast.ForecastDay {
		// We'll use the hourly forecasts for more granular data
		for _, hour := range day.Hour {
			forecastTime := time.Unix(hour.TimeEpoch, 0)

			// Create a forecast entry
			forecast := models.Forecast{
				Temperature: hour.TempC,
				Humidity:    float64(hour.Humidity),
				// Convert from km/h to m/s
				WindSpeed:   hour.WindKph / 3.6,
				WindDeg:     hour.WindDegree,
				Pressure:    hour.PressureMb,
				Description: hour.Condition.Text,
				Icon:        hour.Condition.Icon,
				Timestamp:   forecastTime,
			}

			forecastData.Forecasts = append(forecastData.Forecasts, forecast)
		}
	}

	return forecastData, nil
}
