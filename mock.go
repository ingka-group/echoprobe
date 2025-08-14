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

package echoprobe

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/h2non/gock"
)

// MockCall represents a mocked API call used in tests
type MockCall struct {
	Function func(config *MockConfig)
	Config   *MockConfig
}

// MockConfig configures a Mock
type MockConfig struct {
	Method     string
	StatusCode int
	UrlPath    string
	Response   string
}

// Mock is the struct that gives access to all the mocks
type Mock struct {
	baseURL    string
	httpClient *http.Client
}

// NewMock creates a new Mock
func NewMock(baseURL string) *Mock {
	return &Mock{
		baseURL: baseURL,
	}
}

// TearDown removes all the registered mocks
func (m *Mock) TearDown() {
	gock.Off()
}

// Debug is used to print the request URL and the mock returned for that particular request
func (m *Mock) Debug() {
	gock.Observe(gock.DumpRequest)

	defer func() {
		if !gock.IsDone() {
			fmt.Println("Pending mocks:")
			for _, mock := range gock.Pending() {
				fmt.Printf("- %s %s\n", mock.Request().Method, mock.Request().URLStruct.String())
			}
		}
	}()
}

func (m *Mock) SetHttpClient(httpClient *http.Client) {
	m.httpClient = httpClient
}

func (m *Mock) SetJSON(response *gock.Response, config *MockConfig) {
	var f Fixtures
	if strings.TrimSpace(config.Response) != "" {
		response.JSON(
			f.ReadFixture(
				fmt.Sprintf("%s.json", config.Response),
				"mocks",
			),
		)
	}
}

func (m *Mock) MockRequest(config *MockConfig) {
	if config.StatusCode == 0 {
		config.StatusCode = http.StatusOK
	}

	request := gock.New(m.baseURL)
	if m.httpClient != nil {
		gock.InterceptClient(m.httpClient)
	}

	switch config.Method {
	case http.MethodGet:
		request.Get(config.UrlPath)
	case http.MethodDelete:
		request.Delete(config.UrlPath)
	case http.MethodPost:
		request.Post(config.UrlPath)
	case http.MethodPut:
		request.Put(config.UrlPath)
	case http.MethodPatch:
		request.Patch(config.UrlPath)
	default:
		request.Get(config.UrlPath)
	}

	response := request.Reply(
		config.StatusCode,
	)

	m.SetJSON(response, config)
}
