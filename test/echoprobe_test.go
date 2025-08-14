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
	"net/http"
	"testing"

	"github.com/ingka-group/echoprobe"
)

func TestIntegrationHandler_Ready(t *testing.T) {
	if testing.Short() {
		t.Skip("(skipped)")
	}

	it := echoprobe.NewIntegrationTest(t)
	defer func() {
		it.TearDown()
	}()

	healthHandler := NewHandler()

	tests := []echoprobe.Data{
		{
			Name:       "ok: Ready probe",
			Method:     http.MethodGet,
			Handler:    healthHandler.Ready,
			ExpectCode: http.StatusOK,
		},
	}

	echoprobe.AssertAll(it, tests)
}

func TestIntegrationHandler_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("(skipped)")
	}

	it := echoprobe.NewIntegrationTest(t, echoprobe.IntegrationTestWithPostgres{})
	defer func() {
		it.TearDown()
	}()

	healthHandler := NewHandler()

	tests := []echoprobe.Data{
		{
			Name:           "ok: Live probe",
			Method:         http.MethodGet,
			Handler:        healthHandler.Live,
			ExpectCode:     http.StatusOK,
			ExpectResponse: "live-probe-ok",
		},
	}

	echoprobe.AssertAll(it, tests)
}

func TestIntegrationHandler_MockWeatherWithCustomHttpClient(t *testing.T) {
	if testing.Short() {
		t.Skip("(skipped)")
	}

	// custom http client
	httpClient := http.Client{Transport: &http.Transport{}}

	it := echoprobe.NewIntegrationTest(t, echoprobe.IntegrationTestWithMocks{
		BaseURL: "https://weather.com",
	})
	it.Mock.SetHttpClient(&httpClient)

	handler := NewApiHandler(&httpClient)

	tests := []echoprobe.Data{
		{
			Name:           "ok: Weather forecast",
			Method:         http.MethodGet,
			Handler:        handler.Weather,
			ExpectCode:     http.StatusOK,
			ExpectResponse: "weather-ok",
			Mocks: []echoprobe.MockCall{
				{
					Config: &echoprobe.MockConfig{
						UrlPath:    "/forecast/amsterdam",
						Response:   "weather-ok",
						StatusCode: http.StatusOK,
					},
				},
			},
		},
		{
			Name:           "regression: without a custom client",
			Method:         http.MethodGet,
			Handler:        handler.Weather,
			ExpectCode:     http.StatusOK,
			ExpectResponse: "weather-ok",
			Mocks: []echoprobe.MockCall{
				{
					Config: &echoprobe.MockConfig{
						UrlPath:    "/forecast/amsterdam",
						Response:   "weather-ok",
						StatusCode: http.StatusOK,
					},
				},
			},
		},
	}

	echoprobe.AssertAll(it, tests)
}

func TestIntegrationHandler_MockWeatherWithoutHttpClientConfigured(t *testing.T) {
	if testing.Short() {
		t.Skip("(skipped)")
	}

	it := echoprobe.NewIntegrationTest(t, echoprobe.IntegrationTestWithMocks{
		BaseURL: "https://weather.com",
	})

	httpClient := http.Client{Transport: &http.Transport{}}
	handler := NewApiHandler(&httpClient)

	tests := []echoprobe.Data{
		{
			Name:       "ok: weather forecast mock doesn't work when Mock doesn't contain the custom http.Client",
			Method:     http.MethodGet,
			Handler:    handler.Weather,
			ExpectCode: http.StatusServiceUnavailable,
			Mocks: []echoprobe.MockCall{
				{
					Config: &echoprobe.MockConfig{
						UrlPath:    "/forecast/amsterdam",
						Response:   "weather-ok",
						StatusCode: http.StatusOK,
					},
				},
			},
		},
	}

	echoprobe.AssertAll(it, tests)
}
