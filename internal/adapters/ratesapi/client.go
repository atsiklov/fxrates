package ratesapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	http    *http.Client
	baseURL string
}

func NewClient(httpClient *http.Client, baseURL string) *Client {
	return &Client{http: httpClient, baseURL: baseURL}
}

type ratesResponse struct {
	Result          string             `json:"result"`
	BaseCode        string             `json:"base_code"`
	ConversionRates map[string]float64 `json:"conversion_rates"`
}

func (c *Client) GetExchangeRate(ctx context.Context, code string) (map[string]float64, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}

	// Append the base currency code to the URL path
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
		return nil, fmt.Errorf("rates api status: %s", resp.Status)
	}

	var rr ratesResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return nil, err
	}

	if rr.Result != "success" {
		return nil, fmt.Errorf("api returned non-success result: %s", rr.Result)
	}

	if rr.ConversionRates == nil {
		rr.ConversionRates = map[string]float64{}
	}

	return rr.ConversionRates, nil
}
