package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type ServerRequest struct {
	Server string `json:"server"`
}

type ServersResponse struct {
	Servers []string `json:"servers"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("API request failed with status %d", e.StatusCode)
	}

	return e.Message
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) ListServers(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/dns", nil)
	if err != nil {
		return nil, err
	}

	return c.doServersRequest(req)
}

func (c *Client) AddServer(ctx context.Context, server string) ([]string, error) {
	payload := ServerRequest{
		Server: server,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/dns", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return c.doServersRequest(req)
}

func (c *Client) DeleteServer(ctx context.Context, server string) ([]string, error) {
	endpoint := c.baseURL + "/dns?server=" + url.QueryEscape(server)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return nil, err
	}

	return c.doServersRequest(req)
}

func (c *Client) doServersRequest(req *http.Request) ([]string, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResponse ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Message:    resp.Status,
			}
		}

		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    errorResponse.Error,
		}
	}

	var serversResponse ServersResponse
	if err := json.NewDecoder(resp.Body).Decode(&serversResponse); err != nil {
		return nil, err
	}

	return serversResponse.Servers, nil
}
