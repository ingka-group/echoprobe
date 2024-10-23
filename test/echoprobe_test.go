package test

import (
	"net/http"
	"testing"

	"github.com/ingka-group/echoprobe/src/echoprobe"
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
