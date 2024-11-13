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
	"net/http/httptest"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

const (
	Excel = "xlsx"
)

// Data is a helper struct to define the parameters of a request for a test case.
type Data struct {
	Name               string
	Method             string
	Params             Params
	Handler            func(ctx echo.Context) error
	Mocks              []MockCall
	ExpectResponse     string
	ExpectErrResponse  bool
	ExpectCode         int
	ExpectResponseType string
}

// HandlerResult holds the result of a handler, the error that possibly was returned and the response recorder.
type HandlerResult struct {
	Err      error
	Response *httptest.ResponseRecorder
}

// Assert asserts the result of a handler. Leveraging the HandlerResult struct allows us to assert the
// response of a request by calling the handler function outside the assertion method. Thus, we can pass a custom
// context when calling the handler.
//
// Example:
//
//		for _, t := range tt {
//		  ctx, response := Request(it, t.Method, t.Params)
//
//		  sctx := &CustomContext{
//			Context:   ctx,
//			Clock:     clock.NewMock(),
//		  }
//
//		  err := t.Handler(sctx)
//	   	  if err != nil {
//			it.T.Log(err.Error())
//		  }
//
//		  t.Assert(it, &HandlerResult{
//			err:      err,
//			response: response,
//		  })
//		}
func Assert(it *IntegrationTest, t *Data, res *HandlerResult) {
	it.T.Log(it.T.Name(), "/", t.Name)

	assertHandlerResult(it, t, res)
}

// LoadMocks loads the mocks for a given test case.
func LoadMocks(it *IntegrationTest, t *Data) {
	if it.Mock != nil {
		for i := range t.Mocks {
			it.Mock.MockRequest(t.Mocks[i].Config)
		}
	}
}

// AssertAll runs the given tests and asserts their result. The handler function is called inside the assertion method.
func AssertAll(it *IntegrationTest, tt []Data) {
	for _, t := range tt {
		it.T.Log(it.T.Name(), "/", t.Name)

		LoadMocks(it, &t)

		ctx, response := Request(it, t.Method, t.Params)
		err := t.Handler(ctx)
		if err != nil {
			it.T.Log(err.Error())
		}

		assertHandlerResult(it, &t, &HandlerResult{
			Err:      err,
			Response: response,
		})
	}
}

// assertHandlerResult asserts the result of a handler.
func assertHandlerResult(it *IntegrationTest, t *Data, res *HandlerResult) {
	if t.ExpectErrResponse {
		require.Error(it.T, res.Err)

		echoErr, match := res.Err.(*echo.HTTPError)
		require.True(it.T, match)

		res.Response.Code = echoErr.Code
	} else {
		require.NoError(it.T, res.Err)
	}

	require.Equal(it.T, t.ExpectCode, res.Response.Code)

	if strings.TrimSpace(t.ExpectResponse) != "" {
		if t.ExpectResponseType == Excel {
			expectedRows := it.Fixtures.ReadExcelFile(t.ExpectResponse)

			responseRows, err := it.Fixtures.ExcelToMap(res.Response.Body.Bytes())
			if err != nil {
				it.T.Fatal(err.Error())
			}

			require.Equal(it.T, expectedRows, responseRows)
		} else {
			t.ExpectResponse = it.Fixtures.ReadResponse(t.ExpectResponse)

			require.JSONEq(it.T,
				t.ExpectResponse,
				strings.TrimSpace(res.Response.Body.String()),
			)
		}
	}
}
