package bank

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrBankUnavailable is returned whenever the bank cannot be reached or
// does not give a usable answer (non-2xx status, malformed response,
// timeout). The caller should treat the outcome of the payment as unknown.
var ErrBankUnavailable = errors.New("bank unavailable")

type AuthorizationRequest struct {
	CardNumber string `json:"card_number"`
	ExpiryDate string `json:"expiry_date"`
	Currency   string `json:"currency"`
	Amount     int    `json:"amount"`
	Cvv        string `json:"cvv"`
}

type AuthorizationResponse struct {
	Authorized        bool   `json:"authorized"`
	AuthorizationCode string `json:"authorization_code"`
}

type Authorizer interface {
	Authorize(ctx context.Context, req AuthorizationRequest) (AuthorizationResponse, error)
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

var _ Authorizer = (*Client)(nil)

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Authorize(ctx context.Context, req AuthorizationRequest) (AuthorizationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return AuthorizationResponse{}, fmt.Errorf("marshal authorization request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/payments", bytes.NewReader(body))
	if err != nil {
		return AuthorizationResponse{}, fmt.Errorf("build authorization request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return AuthorizationResponse{}, ErrBankUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AuthorizationResponse{}, ErrBankUnavailable
	}

	var authResp AuthorizationResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return AuthorizationResponse{}, ErrBankUnavailable
	}

	return authResp, nil
}
