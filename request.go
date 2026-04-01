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
	"io"
	"mime/multipart"
	"net/http/httptest"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

// FileUpload defines the file to be uploaded in a multipart form request.
type FileUpload struct {
	FieldName string
	Fixture   string
}

// Params define the parameters of a request.
type Params struct {
	Path  map[string]string
	Query map[string][]string
	Body  string
	Form  *Form
}

// Form defines the parameters for a multipart form request.
type Form struct {
	File   *FileUpload // file to upload in a multipart form request
	Fields map[string]string
}

// Request creates a new request and a new test service context to which it passes the required parameters.
func Request(it *IntegrationTest, method string, params Params) (echo.Context, *httptest.ResponseRecorder) {
	var reader io.Reader
	var contentType string

	// Validate that multipart form and body are not combined
	hasMultipart := params.Form != nil
	hasBody := strings.TrimSpace(params.Body) != ""
	if hasMultipart && hasBody {
		it.T.Fatalf("echoprobe: Request failed: cannot combine multipart form with body")
	}

	if hasMultipart {
		reader, contentType = handleMultipart(it, params.Form)
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

	return ctx, response
}

// handleMultipart creates a multipart form body from the provided form parameters.
func handleMultipart(it *IntegrationTest, form *Form) (io.Reader, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if form.File != nil {
		// Get file content from fixture
		var fileContent []byte
		if form.File.Fixture != "" {
			fileContent = it.Fixtures.ReadFileBytes(form.File.Fixture)
		} else {
			it.T.Fatalf("echoprobe: Request failed: no fixture provided for file upload")
		}

		// Default field name to "file" if not provided
		fieldName := strings.TrimSpace(form.File.FieldName)
		if fieldName == "" {
			fieldName = "file"
		}

		// Create form file
		part, err := writer.CreateFormFile(fieldName, filepath.Base(form.File.Fixture))
		if err != nil {
			it.T.Fatalf("echoprobe: Request failed to create form file: %v", err)
		}
		if _, err = part.Write(fileContent); err != nil {
			it.T.Fatalf("echoprobe: Request failed to write file content: %v", err)
		}
	}

	if form.Fields != nil {
		for key, value := range form.Fields {
			if err := writer.WriteField(key, value); err != nil {
				it.T.Fatalf("echoprobe: Request failed to write field %s: %v", key, err)
			}
		}
	}

	if err := writer.Close(); err != nil {
		it.T.Fatalf("echoprobe: Request failed to close multipart writer: %v", err)
	}

	return body, writer.FormDataContentType()
}
