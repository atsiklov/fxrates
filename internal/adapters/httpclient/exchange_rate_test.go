package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExchangeRateClient_Success(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
            "result": "success",
            "base_code": "USD",
            "conversion_rates": {"EUR": 0.92, "JPY": 150.0}
        }`))
	}))
	t.Cleanup(srv.Close)

	baseURL := srv.URL + "/api/latest/"
	c := NewExchangeRateClient(srv.Client(), baseURL)

	ratesMap, err := c.GetExchangeRates(context.Background(), "USD")
	require.NoError(t, err)
	require.Equal(t, "/api/latest/USD", gotPath)
	require.Len(t, ratesMap, 2)
	require.InDelta(t, 0.92, ratesMap["EUR"], 1e-9)
	require.InDelta(t, 150.0, ratesMap["JPY"], 1e-9)
}

func TestExchangeRateClient_StatusCodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	c := NewExchangeRateClient(srv.Client(), srv.URL+"/latest")

	_, err := c.GetExchangeRates(context.Background(), "USD")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected status code 503")
	require.Contains(t, err.Error(), "USD")
}

func TestExchangeRateClient_JSONDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{")) // invalid JSON
	}))
	t.Cleanup(srv.Close)

	c := NewExchangeRateClient(srv.Client(), srv.URL+"/latest")

	_, err := c.GetExchangeRates(context.Background(), "USD")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to decode response for currency \"USD\"")
}

func TestExchangeRateClient_NonSuccessResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": "error", "base_code": "USD", "conversion_rates": {}}`))
	}))
	t.Cleanup(srv.Close)

	c := NewExchangeRateClient(srv.Client(), srv.URL+"/latest")

	_, err := c.GetExchangeRates(context.Background(), "USD")
	require.Error(t, err)
	require.Contains(t, err.Error(), "api returned non-success result for currency \"USD\": error")
}

func TestExchangeRateClient_BaseURLParseError(t *testing.T) {
	c := NewExchangeRateClient(&http.Client{}, "http://::1]")
	_, err := c.GetExchangeRates(context.Background(), "USD")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse base URL")
}
