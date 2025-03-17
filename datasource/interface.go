package datasource

import (
	"context"

	"weather-service/models"
)

// DataSource defines the interface for any weather data provider
type DataSource interface {
	Name() string
	FetchWeatherData(ctx context.Context, location string) (models.WeatherData, error)
}
