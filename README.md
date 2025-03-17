# Weather Service

A Go-based weather service that aggregates data from multiple weather providers and exposes it through a REST API.

## Features

- **Multi-Provider Support**: Get weather data from multiple providers including OpenWeatherMap and WeatherAPI
- **Current Weather**: Fetch current weather conditions for any location
- **Weather Forecasts**: Get weather forecasts for up to 5 days
- **REST API**: Easy-to-use REST endpoints to access weather data
- **Provider Aggregation**: Compare weather data from different providers
- **Automatic Updates**: Regular background updates of weather data
- **Caching**: In-memory caching of weather data
- **Forecast Support**: Retrieves and stores weather forecasts
- **Rate Limiting**: Token bucket rate limiter to respect API provider rate limits

## Rate Limiting

The service implements a token bucket rate limiter for API providers to ensure we respect their rate limits. This is particularly important for free tier API keys which often have strict rate limits.

### How the Rate Limiter Works

The rate limiter uses the token bucket algorithm implemented via the `golang.org/x/time/rate` package:

1. Each provider is wrapped with a rate limiter
2. The rate limiter controls how many requests can be made per second
3. It supports "burst" scenarios where a limited number of requests can exceed the rate limit
4. Requests that would exceed the rate limit are delayed until tokens are available

### Configuration

Rate limiting is enabled by default and can be controlled via the `-rate-limit` flag (set to `false` to disable).

Rate limits are set per provider based on their documented API limits:
- OpenWeatherMap (free tier): 60 calls/minute = 1 request per second
- WeatherAPI (free tier): ~23 calls/minute = 0.4 requests per second

## Configuration

The service is configured via a `config.json` file with the following structure:

```json
{
  "openWeatherMap": {
    "enabled": true,
    "apiKey": "your-openweathermap-api-key"
  },
  "weatherAPI": {
    "enabled": true,
    "apiKey": "your-weatherapi-key"
  },
  "locations": [
    "London,UK",
    "New York,US",
    "Tokyo,JP"
  ]
}
```

## API Endpoints

### Current Weather

- **GET /api/weather/locations**: Get a list of all available locations
- **GET /api/weather/location/{location}**: Get current weather data for a specific location

### Forecasts

- **GET /api/forecast/location/{location}**: Get forecast data for a specific location
- **GET /api/forecast/location/{location}?days=3**: Specify number of days (1-5) for the forecast
- **GET /api/forecast/location/{location}/{provider}**: Get forecast from a specific provider

### System

- **GET /api/health**: Health check endpoint

## Running the Service

### Prerequisites

- Go 1.19 or higher
- API keys for weather providers (set in config.json)

### Installation

```bash
go mod tidy
go build
```

### Configuration

Create a `config.json` file:

```json
{
  "open_weather_map": {
    "enabled": true,
    "api_key": "YOUR_OWM_API_KEY"
  },
  "weather_api": {
    "enabled": true,
    "api_key": "YOUR_WEATHERAPI_KEY"
  },
  "locations": [
    "London,GB",
    "New York,US",
    "Tokyo,JP"
  ]
}
```

### Running the Service

```bash
./weather-service -port=8080 -update=5m -config=config.json
```

Options:
- `-port`: Port to run the server on (default: 8080)
- `-update`: Weather data update interval (default: 5m)
- `-config`: Path to configuration file (default: config.json)
- `-rate-limit`: Enable API rate limiting (default: true)

## Testing the Rate Limiter

A test utility is included to verify rate limiting behavior:

```bash
go build -o rate_limit_test cmd/rate_limit_test/main.go
./rate_limit_test -rps=1.0 -burst=3 -requests=10 -concurrent=5
```

Options:
- `-rps`: Requests per second limit (default: 1.0)
- `-burst`: Maximum burst size (default: 3)
- `-requests`: Total number of requests to make (default: 10)
- `-concurrent`: Number of concurrent workers (default: 5)

## Building

```bash
go build -o weather-service
```

## Example API Response

### Current Weather

```json
{
  "location": "London,GB",
  "data": [
    {
      "provider": "OpenWeatherMap",
      "location": "London,GB",
      "temperature": 15.3,
      "humidity": 76,
      "windSpeed": 4.1,
      "pressure": 1012,
      "description": "scattered clouds",
      "icon": "03d",
      "windDeg": 260,
      "timestamp": "2023-07-05T12:30:00Z"
    },
    {
      "provider": "WeatherAPI",
      "location": "London,GB",
      "temperature": 15.5,
      "humidity": 77,
      "windSpeed": 4.2,
      "pressure": 1012.5,
      "description": "Partly cloudy",
      "icon": "https://cdn.weatherapi.com/weather/64x64/day/116.png",
      "windDeg": 255,
      "timestamp": "2023-07-05T12:30:00Z"
    }
  ],
  "timestamp": "2023-07-05T12:30:00Z"
}
```

### Forecast

```json
{
  "location": "London,GB",
  "forecasts": [
    {
      "provider": "OpenWeatherMap",
      "location": "London,GB",
      "forecasts": [
        {
          "temperature": 15.3,
          "humidity": 76,
          "windSpeed": 4.1,
          "windDeg": 260,
          "pressure": 1012,
          "description": "scattered clouds",
          "icon": "03d",
          "timestamp": "2023-07-05T15:00:00Z"
        },
        {
          "temperature": 14.7,
          "humidity": 78,
          "windSpeed": 3.8,
          "windDeg": 255,
          "pressure": 1013,
          "description": "light rain",
          "icon": "10d",
          "timestamp": "2023-07-05T18:00:00Z"
        }
        // More forecast data points...
      ],
      "updated": "2023-07-05T12:30:00Z"
    }
  ],
  "timestamp": "2023-07-05T12:30:00Z"
}
```

## Dependencies

- Go 1.16 or later

## License

MIT

## Docker Support

### Building the Docker Image

```bash
docker build -t weather-service .
```

### Running with Docker

```bash
# Run with default settings
docker run -p 8080:8080 --name weather-service weather-service

# Run with environment variables
docker run -p 8080:8080 \
  -e OPENWEATHERMAP_API_KEY=your_key_here \
  -e WEATHERAPI_KEY=your_key_here \
  -e CONFIG_FILE=config.json \
  -e ENABLE_RATE_LIMIT=true \
  --name weather-service weather-service

# Run with custom config mounted from host
docker run -p 8080:8080 \
  -v $(pwd)/config.json:/app/config.json:ro \
  --name weather-service weather-service
  
# Disable rate limiting
docker run -p 8080:8080 \
  -e ENABLE_RATE_LIMIT=false \
  --name weather-service weather-service
```

### Using Docker Compose

1. Set up your environment variables:
   ```bash
   # Copy example file
   cp .env.example .env
   
   # Edit with your API keys
   vi .env
   ```

2. (Optional) Create a custom override file for Docker Compose:
   ```bash
   # Copy the example override file
   cp docker-compose.override.yml.example docker-compose.override.yml
   
   # Edit to customize your deployment
   vi docker-compose.override.yml
   ```

3. Start the service:
   ```bash
   docker-compose up -d
   ```

4. View logs:
   ```bash
   docker-compose logs -f
   ```

5. Stop the service:
   ```bash
   docker-compose down
   ``` 