// Copyright © 2024 Ingka Holding B.V. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// ServiceHealth defines the health of the service.
type ServiceHealth struct {
	ServiceStatus ServiceHealthStatus      `json:"status"`
	Description   ServiceHealthDescription `json:"description"`
	CompletedAt   time.Time                `json:"completed_at"`
}

// ServiceHealthStatus defines the status of the service.
type ServiceHealthStatus string // @name serviceHealthStatus

const (
	statusHealthy ServiceHealthStatus = "healthy"
)

// ServiceHealthDescription describes the state of the service status.
type ServiceHealthDescription string // @name serviceHealthDescription

const (
	descriptionHealthy ServiceHealthDescription = "everything is awesome"
)

// Handler defines the http router implementation for health endpoints.
type Handler struct{}

// NewHandler creates a new Handler for health endpoints.
func NewHandler() *Handler {
	return &Handler{}
}

// Ready performs readiness check.
//
// @Summary Ready healthcheck
// @Description Performs readiness check
// @Tags health
// @ID health-ready
// @Success 200 "OK"
// @Failure 503 {object} ServiceHealth "Service Unavailable"
// @Router /health/ready [get]
func (h *Handler) Ready(ctx echo.Context) error {
	return ctx.NoContent(http.StatusOK)
}

// Live performs a live check.
//
// @Summary Live healthcheck
// @Description Performs a live check
// @Tags health
// @ID health-live
// @Produce json
// @Success 200 {object} ServiceHealth "OK"
// @Failure 503 {object} ServiceHealth "Service Unavailable"
// @Router /health/live [get]
func (h *Handler) Live(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, ServiceHealth{
		ServiceStatus: statusHealthy,
		Description:   descriptionHealthy,
	})
}

// WeatherForecast provides information about the weather
type WeatherForecast struct {
	Location ForecastLocation `json:"location"`
	Summary  ForecastSummary  `json:"summary"`
} // @name WeatherForecast

// ForecastLocation defines the location of the weather forecast.
type ForecastLocation string // @name forecastLocation

const (
	forecastLocation ForecastLocation = "Amsterdam"
)

// ForecastSummary the specifics of the weather forecast.
type ForecastSummary string // @name forecastSummary

const (
	forecastSummary ForecastSummary = "Sunny, between 30 and 15 degrees celsius"
	failedSummary   ForecastSummary = "Summary could not be loaded."
)

type ApiHandler struct {
	HttpClient *http.Client
}

func NewApiHandler(httpClient *http.Client) *ApiHandler {
	return &ApiHandler{
		HttpClient: httpClient,
	}
}

// Weather fetches the latest weather forecast for Amsterdam.
//
// @Summary Weather forecast
// @Description Performs an external api call to get weather forecast
// @Tags weather
// @ID weather-forecast
// @Produce json
// @Success 200 {object} WeatherForecast "OK"
// @Failure 503 {object} WeatherForecast "Service Unavailable"
// @Router /weather/amsterdam [get]
func (h *ApiHandler) Weather(ctx echo.Context) error {
	res, err := h.HttpClient.Get("https://weather.com/forecast/amsterdam")
	if err != nil {
		fmt.Println(err)
		return ctx.JSON(http.StatusServiceUnavailable, WeatherForecast{
			Location: forecastLocation,
			Summary:  failedSummary,
		})
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return ctx.JSON(http.StatusServiceUnavailable, WeatherForecast{
			Location: forecastLocation,
			Summary:  failedSummary,
		})
	}

	if res.StatusCode != http.StatusOK {
		fmt.Printf("Weather API returned status: %d\n", res.StatusCode)
		return ctx.JSON(http.StatusServiceUnavailable, WeatherForecast{
			Location: forecastLocation,
			Summary:  failedSummary,
		})
	}

	var weatherData map[string]any
	if err := json.Unmarshal(body, &weatherData); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return ctx.JSON(http.StatusServiceUnavailable, WeatherForecast{
			Location: forecastLocation,
			Summary:  failedSummary,
		})
	}

	return ctx.JSON(http.StatusOK, weatherData)
}
