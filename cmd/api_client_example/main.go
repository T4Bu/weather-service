package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	fmt.Println("Weather API Client Example")
	fmt.Println("=========================")

	// Base URL for the API
	baseURL := "http://localhost:8080"

	// Wait a moment for the server to fetch some data
	fmt.Println("Waiting for weather service to collect initial data...")
	time.Sleep(5 * time.Second)

	// Get available locations
	fmt.Println("\nFetching available locations...")
	locationsURL := fmt.Sprintf("%s/api/weather/locations", baseURL)
	locationsResp, err := http.Get(locationsURL)
	if err != nil {
		fmt.Printf("Error fetching locations: %v\n", err)
		os.Exit(1)
	}
	defer locationsResp.Body.Close()

	var locationsData map[string]interface{}
	locationsBody, _ := io.ReadAll(locationsResp.Body)
	json.Unmarshal(locationsBody, &locationsData)

	fmt.Printf("Available locations: %v\n\n", locationsData["locations"])

	// Choose a location to query (if available)
	var locations []interface{}
	if locs, ok := locationsData["locations"].([]interface{}); ok {
		locations = locs
	}

	if len(locations) == 0 {
		fmt.Println("No locations available yet. Try again later.")
		return
	}

	// Get the first location from the list
	location := locations[0].(string)
	fmt.Printf("Fetching weather data for %s...\n", location)

	// Get weather for the selected location
	weatherURL := fmt.Sprintf("%s/api/weather/location/%s", baseURL, location)
	weatherResp, err := http.Get(weatherURL)
	if err != nil {
		fmt.Printf("Error fetching weather: %v\n", err)
		os.Exit(1)
	}
	defer weatherResp.Body.Close()

	// Read and pretty print the response
	weatherBody, _ := io.ReadAll(weatherResp.Body)

	// Parse the JSON for pretty printing
	var weatherData map[string]interface{}
	json.Unmarshal(weatherBody, &weatherData)

	// Pretty print the result
	prettyJSON, _ := json.MarshalIndent(weatherData, "", "  ")
	fmt.Printf("\nWeather data for %s:\n%s\n", location, string(prettyJSON))
}
