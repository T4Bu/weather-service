package models

import (
	"time"
)

// WeatherData represents the weather data from a provider
type WeatherData struct {
	Provider    string    `json:"provider"`
	Location    string    `json:"location"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	WindSpeed   float64   `json:"windSpeed"`
	Pressure    float64   `json:"pressure"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	WindDeg     int       `json:"windDeg"`
	Timestamp   time.Time `json:"timestamp"`
}
