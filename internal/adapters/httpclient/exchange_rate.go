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

func (c *ExchangeRateClient) GetExchangeRates(ctx context.Context, code string) (map[string]float64, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}

	u.Path = strings.TrimSuffix(u.Path, "/") + "/" + code

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("rate api status: %s", resp.Status) // todo: add custom error
	}

	var body apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	if body.Result != "success" {
		return nil, fmt.Errorf("api returned non-success result: %s", body.Result) // todo: add custom error
	}

	if body.ConversionRates == nil {
		return nil, fmt.Errorf("api returned non-success result: %s", body.Result) // todo: add custom error
	}

	return body.ConversionRates, nil
}

func NewExchangeRateClient(httpClient *http.Client, baseURL string) *ExchangeRateClient {
	return &ExchangeRateClient{http: httpClient, baseURL: baseURL}
}
