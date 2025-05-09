# echoprobe 🧪

**echoprobe** is a simple Go library for writing integration tests in for [echo](https://github.com/labstack/echo) framework.
It uses [test-containers](https://golang.testcontainers.org/) and [gock](https://github.com/h2non/gock) to provide a simple
and easy-to-use interface to write tests. supporting real HTTP requests and responses in various formats.


## Features

- Real HTTP requests and responses in `JSON` and `Excel` formats
- Mocking of external HTTP calls
- Database integrations with `PostgreSQL` and `BigQuery`
- Fully compatible with [fastecho](https://github.com/ingka-group/fastecho)


## Get Started

Install `echoprobe`, using `go get`:

```bash
$ go get github.com/ingka-group/echoprobe
```

Below you can find a list of examples in order to use `echoprobe`.
- [Basic usage](#basic-usage)
- [With PostgreSQL](#with-postgresql)
- [With BigQuery](#with-bigquery)
- [With Mocks](#with-mocks)
- [With PostgreSQL and Mocks](#with-postgresql-and-mocks)
- [With Excel](#with-excel)
- [Error responses](#error-responses)
- [Query parameters](#query-parameters)
- [Request body](#request-body)
- [Assert with custom context](#assert-with-custom-context)

You can find some complete examples in the [test](./test) directory.

### Basic usage

To write an integration test, you need to create a new test file with the `_test.go` suffix. In case your test cases of your handlers expect a response, you can leverage the `fixtures` feature. Fixtures are optional, and are required only in case you expect a response body. A `fixtures` folder has to exist in the location where the `_test.go` files are so that files can be read. This also ensures that fixtures are kept close to the test and in the relevant package.
To store the expected responses, you need to create a `responses` directory under `fixtures` to store your JSON responses. For example, `fixtures/responses/my_response.json`.

```golang
it := echoprobe.NewIntegrationTest(
    t,
)
defer func() {
    it.TearDown()
}()

handler := NewHandler()

tests := []echoprobe.Data{
    {
        Name:   "ok: my test case",
        Method: http.MethodGet,
        Params: echoprobe.Params {
            Path: map[string]string {
                "id": "1",
            },
        },
        Handler:        handler.MyEndpoint,
        ExpectCode:     http.StatusOK,
        ExpectResponse: "my_response",
    },
}

echoprobe.AssertAll(it, tests)
```

### With PostgreSQL

To use PostgreSQL in your integration test, you need to pass the `IntegrationTestWithPostgres` option to the `NewIntegrationTest` function. _Optionally_, you can initialize your database using a SQL script, that will be executed before the test starts. The script should contain the necessary DDL and DML statements to prepare the database for the test. The script must be present under `fixtures`. For example, `fixtures/init-db.sql`.

```golang
it := echoprobe.NewIntegrationTest(
    t,
    echoprobe.IntegrationTestWithPostgres{
        InitSQLScript: "init-db.sql",
    },
)

defer func() {
    it.TearDown()
}()

repository := NewRepository(it.Db)
service :=    NewService(repository)
handler :=    NewHandler(service)

tests := []echoprobe.Data{...}

echoprobe.AssertAll(it, tests)
```

### With BigQuery

`echoprobe` supports testing with BigQuery using `ghcr.io/goccy/bigquery-emulator` as a test contair. To use BigQuery in your integration test, you need to pass the `IntegrationTestWithBigQuery` option to the `NewIntegrationTest` function. It is expected that BigQuery needs to be populated with data upon the test startup. To do that, you need to provide a `.yaml` under the `fixtures/bigquery` directory.
The YAML file should contain the necessary format so that BigQuery emulator can mount the data in the container.

```golang

it := echoprobe.NewIntegrationTest(
    t,
    echoprobe.IntegrationTestWithBigQuery{
        DataPath: "/fixtures/bigquery/data.yaml",
    },
)

bqClient, err := NewBigQueryClient(it.BigQuery)

repository := NewRepository(bqClient)
service :=    NewService(repository)
handler:=     NewHandler(service)

tests := []echoprobe.Data{...}

echoprobe.AssertAll(it, tests)

func NewBigQueryClient(t *testing.T, bq *echoprobe.BigqueryEmulatorContainer) (*bigquery.Client, error) {
    client, err := bigquery.NewClient(
        context.Background(),
        "test",
        option.WithoutAuthentication(),
        option.WithEndpoint(fmt.Sprintf("http://%s:%d", bq.BqHost, bq.BqRestPort)),
    )

    defer func(client *bigquery.Client) {
        err := client.Close()
        if err != nil {
            t.Fatalf("unable to close big query client: %v", err)
        }
    }(client)

    return client, err
}
```

### With Mocks

Mock responses are optional and must be stored with the rest of the fixtures as `.json` files in a `mocks` folder within `fixtures`. For example, a mock called `my_mock.json` would be stored in `fixtures/mocks/my_mock.json`. Mocking a request, consists of pairing a request URL with a status code and optionally a response.

**NOTE**: Mocks are single-use. If your code repeatedly calls the same endpoint expecting the same response, writing this once is not enough. You need to replicate the mock itself to make a request reuse a mock a specific number of times.

```golang
it := echoprobe.NewIntegrationTest(
    t,
    echoprobe.IntegrationTestWithMocks{
        BaseURL: "/v1",
    },
)
defer func() {
    it.TearDown()
}()

handler := NewHandler()

tests := []echoprobe.Data{
    {
        Name:   "ok: my test case",
        Method: http.MethodGet,
        Params: echoprobe.Params {
            Path: map[string]string {
                "id": "1",
            },
        },
        Mocks: []echoprobe.MockCall{
            {
                Config: &echoprobe.MockConfig{
                    UrlPath:    fmt.Sprintf("/v1/users/%s", "1"),
                    Response:   "my_mock",
                    StatusCode: http.StatusOK,
                },
            },
        },
        Handler:        handler.MyEndpoint,
        ExpectCode:     http.StatusOK,
        ExpectResponse: "my_response",
    },
}

echoprobe.AssertAll(it, tests)
```

### With PostgreSQL and Mocks

An integration test support various types of features all at once. In order to use PostgreSQL and Mocks, you can use the following example. Similarly, you can append other options to your integration test.

```golang
it := echoprobe.NewIntegrationTest(
    t,
    echoprobe.IntegrationTestWithMocks{
        BaseURL: "/v1",
    },
    echoprobe.IntegrationTestWithPostgres{},
)
```

### With Excel

`echoprobe` supports testing with Excel files. To compare the result of a handler with the expected Excel file, you need to store the Excel file(s) under `excel` in the `fixtures` folder. For example, `fixtures/excel/my_excel.xlsx`.
Additionally, you need to pass some extra instructions to the test case, to identify that the response is expected to be an Excel file.

```golang
tests := []echoprobe.Data{
    {
        Name:   "ok: my test case",
        Method: http.MethodGet,
        Params: echoprobe.Params {
            Path: map[string]string {
                "id": "1",
            },
        },
        Handler:            handler.MyEndpoint,
        ExpectCode:         http.StatusOK,
        ExpectResponseType: echoprobe.Excel,
        ExpectResponse:     "my_excel",
    },
}
```

### Error responses

`echoprobe` is fully compatible with the `echo.NewHTTPError` response, meaning that you can test error responses as well
by setting the `ExpectErrResponse` field to `true` in the test data.

```golang
tests := []echoprobe.Data{
    {
        Name:   "error: my test case",
        Method: http.MethodGet,
        Params: echoprobe.Params {
            Path: map[string]string {
                "id": "INVALID_ID",
            },
        },
        Handler:           handler.MyEndpoint,
        ExpectCode:        http.StatusBadRequest,
        ExpectErrResponse: true,
    },
}
```

### Query parameters

`echoprobe` supports also query parameters in the request. You can pass them in the `Params` of the `Data` struct. The structure supports multiple query parameters with the same key to cover situations where the endpoint supports multiple values for the same parameter.

```golang
tests := []echoprobe.Data{
    {
        Name:   "ok: my test case",
        Method: http.MethodGet,
        Params: echoprobe.Params {
            Body: "my_body",
            Query: map[string][]string {
                "param1": {"value1", "value2"},
            }
        },
        Handler:    handler.MyEndpoint,
        ExpectCode: http.StatusNoContent,
    },
}
```

### Request body

In case your request required a body, you can pass it in the `Data` struct. In such cases, you need to store the JSON of the request body under `requests` in the `fixtures` folder. For example, `fixtures/requests/my_body.json`.

```golang
tests := []echoprobe.Data{
    {
        Name:   "ok: my test case",
        Method: http.MethodPost,
        Params: echoprobe.Params {
            Body: "my_body",
        },
        Handler:        handler.MyEndpoint,
        ExpectResponse: "my_response",
        ExpectCode:     http.StatusCreated,
    },
}
````

### Assert with custom context

`echoprobe` provides the function `AssertAll` to assert the test cases. In case you need to assert the test cases with a custom context, you can create your own function for that.

```golang
// AssertAllWithCustomContext is a helper function to run multiple tests in a single test function.
// Before asserting a test, the function prepares the custom context and calls the handler function.
func AssertAllWithCustomContext(it *echoprobe.IntegrationTest, tt []echoprobe.Data) {
    for _, t := range tt {
        ctx, response := echoprobe.Request(it, t.Method, t.Params)

        // Create the custom context
        sctx := &CustomContext{
            Context:   ctx,
            Clock:     clock.NewMock(),
        }

        // Attach the custom context to the handler
        err := t.Handler(sctx)
        if err != nil {
            it.T.Log(err.Error())
        }

        // Assert the test
        echoprobe.Assert(it, &t, &echoprobe.HandlerResult{
            Err:      err,
            Response: response,
        })
    }
}
```

### Testing

To run the full set of tests you can execute the following command.

```bash
$ make test
```

### Pre-commit

This project uses pre-commit(https://pre-commit.com/) to integrate code checks used to gate commits.

```bash
# required only once
$ pre-commit install
pre-commit installed at .git/hooks/pre-commit

# run checks on all files
$ make pre-commit
```

### Other management tasks

For more information on the available targets, you can access the inline help of the Makefile.

```bash
$ make help
```

or, equivalently,

```bash
$ make
```


## Contributing
Please read [CONTRIBUTING](./CONTRIBUTING.md) for more details about making a contribution to this open source project and ensure that you follow our [CODE_OF_CONDUCT](./CODE_OF_CONDUCT.md).


## Contact
If you have any other issues or questions regarding this project, feel free to contact one of the [code owners/maintainers](.github/CODEOWNERS) for a more in-depth discussion.


## Licence
This open source project is licensed under the "Apache-2.0", read the [LICENCE](./LICENCE.md) terms for more details.
