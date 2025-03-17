package models

import (
	"time"
)

// Forecast represents a single forecast point with weather conditions at a specific time
type Forecast struct {
	Temperature float64   `json:"temperature"` // in Celsius
	Humidity    float64   `json:"humidity"`    // percentage
	WindSpeed   float64   `json:"windSpeed"`   // in m/s
	WindDeg     int       `json:"windDeg"`     // wind direction in degrees
	Pressure    float64   `json:"pressure"`    // in hPa
	Description string    `json:"description"` // short text description
	Icon        string    `json:"icon"`        // icon code or URL
	Timestamp   time.Time `json:"timestamp"`   // time this forecast is for
}

// ForecastData represents weather forecast data from a provider
type ForecastData struct {
	Provider  string     `json:"provider"`  // weather data provider name
	Location  string     `json:"location"`  // location name
	Forecasts []Forecast `json:"forecasts"` // list of forecasts
	Updated   time.Time  `json:"updated"`   // when this forecast was updated
}
