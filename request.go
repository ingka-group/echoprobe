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
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"

	"github.com/labstack/echo/v4"
)

// FileUpload defines the file to be uploaded in a multipart form request.
type FileUpload struct {
	FieldName string // the form field name for the file
	Fixture   string // the fixture file path (if loading from fixtures)
}

// Params define the parameters of a request.
type Params struct {
	Path  map[string]string
	Query map[string][]string
	Body  string
	File  *FileUpload // file to upload in a multipart form request
}

// Request creates a new request and a new test service context to which it passes the required parameters.
func Request(it *IntegrationTest, method string, params Params) (echo.Context, *httptest.ResponseRecorder, error) {
	var reader io.Reader
	var contentType string

	// Handle file upload with multipart form
	if params.File != nil {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Get file content from fixture or direct content
		var fileContent []byte
		if params.File.Fixture != "" {
			fileContent = it.Fixtures.ReadFileBytes(params.File.Fixture)
		} else {
			return nil, nil, fmt.Errorf("file content is required for file upload")
		}

		// Create form file
		part, err := writer.CreateFormFile(params.File.FieldName, params.File.Fixture)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create form file: %w", err)
		}
		_, err = part.Write(fileContent)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to write file content: %w", err)
		}

		// Add body fields if present
		if strings.TrimSpace(params.Body) != "" {
			params.Body = it.Fixtures.ReadRequestBody(params.Body)
			err = writer.WriteField("body", params.Body)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to write body field: %w", err)
			}
		}

		err = writer.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to close multipart writer: %w", err)
		}

		reader = body
		contentType = writer.FormDataContentType()
	} else {
		// If the body is not empty, read the body fixture and create a reader from it.
		// NOTE: The body expects the filename of the fixture, not the content.
		if strings.TrimSpace(params.Body) != "" {
			params.Body = it.Fixtures.ReadRequestBody(params.Body)
			reader = strings.NewReader(params.Body)
		}
		contentType = echo.MIMEApplicationJSON
	}

	// 2nd parameter is supposed to be the URI but since we inject everything via context, we can ignore this
	req := httptest.NewRequest(method, "/", reader)
	req.Header.Set(
		echo.HeaderContentType,
		contentType,
	)

	response := httptest.NewRecorder()
	ctx := it.Echo.NewContext(req, response)

	if params.Path != nil {
		var paramNames []string
		var paramValues []string

		for name, value := range params.Path {
			paramNames = append(paramNames, name)
			paramValues = append(paramValues, value)
		}

		ctx.SetParamNames(paramNames...)
		ctx.SetParamValues(paramValues...)
	}

	// params.Query is a map with value as a slice of strings
	// This is required in case we want to pass multiple values for
	// the same query parameter. For example /v1/sales?status=active&status=inactive
	if params.Query != nil {
		q := ctx.QueryParams()
		for name, value := range params.Query {
			for i := range value {
				q.Add(name, value[i])
			}
		}
	}

	return ctx, response, nil
}
