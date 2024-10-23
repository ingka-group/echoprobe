# echoprobe

**echoprobe** is a simple Go library for writing integration tests in for [echo](https://github.com/labstack/echo) framework.
It uses [test-containers](https://golang.testcontainers.org/) and [gock](https://github.com/h2non/gock) to provide a simple
and easy-to-use interface to write tests. supporting real HTTP requests and responses in various formats.


## Features

- Real HTTP requests and responses in `JSON` and `Excel` formats
- Mocking of external HTTP calls
- Database integrations with `PostgreSQL` and `BigQuery`
- Fully compatible with [fastecho](https://github.com/ingka-group/fastecho)


## Get Started (WIP)

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


### Basic usage

TBD
Fixtures are optional, and are required only in case you expect a response body.
A `fixtures` folder has to exist in the location where the `_test.go` files are so that files can be read. This also ensures that fixtures are kept close to the test and in the relevant package.


### With PostgreSQL

TBD

### With BigQuery

TBD

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

handler := myHandler()

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
					Reponse:    "my_mock.json",
                    StatusCode: http.StatusNotFound,
                },
            },
        },
        Handler:    handler.MyEndpoint,
        ExpectCode: http.StatusOK,
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
        ExpectResponse:     "my_excel.xlsx",
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
            Body: "my_body.json", 
            Query: map[string][]string {
                "param1": {"value1", "value2"},
            }
        },
        Handler:    handler.MyEndpoint,
        ExpectCode: http.StatusOK,
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
            Body: "my_body.json",
        },
        Handler:    handler.MyEndpoint,
        ExpectCode: http.StatusCreated,
    },
}
````

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
