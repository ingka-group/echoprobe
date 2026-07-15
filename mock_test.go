package echoprobe

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegrationTestWithMocksInterceptsClientsCreatedBeforeMocksAreLoaded(t *testing.T) {
	it := NewIntegrationTest(t, IntegrationTestWithMocks{BaseURL: "http://echoprobe.invalid"})
	t.Cleanup(it.TearDown)

	client := &http.Client{Transport: http.DefaultTransport}
	it.Mock.MockRequest(&MockConfig{
		Method:     http.MethodGet,
		UrlPath:    "/resource",
		StatusCode: http.StatusNoContent,
	})

	resp, err := client.Get("http://echoprobe.invalid/resource")
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
}
