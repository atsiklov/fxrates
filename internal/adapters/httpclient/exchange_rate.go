package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type ExchangeRateClient struct {
	http    *http.Client
	baseURL string
}

type apiResponse struct {
	Result          string             `json:"result"`
	BaseCode        string             `json:"base_code"`
	ConversionRates map[string]float64 `json:"conversion_rates"`
}

func (c *ExchangeRateClient) GetExchangeRates(ctx context.Context, base string) (map[string]float64, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	u.Path = strings.TrimSuffix(u.Path, "/") + "/" + base

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for currency %q: %w", base, err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request for currency %q: %w", base, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status base %d for currency %q: %s", resp.StatusCode, base, resp.Status)
	}

	var body apiResponse
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode response for currency %q: %w", base, err)
	}

	if body.Result != "success" {
		return nil, fmt.Errorf("api returned non-success result for currency %q: %s", base, body.Result)
	}

	return body.ConversionRates, nil
}

func NewExchangeRateClient(httpClient *http.Client, baseURL string) *ExchangeRateClient {
	return &ExchangeRateClient{http: httpClient, baseURL: baseURL}
}
