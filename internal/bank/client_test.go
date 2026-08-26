package bank

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Authorize(t *testing.T) {
	t.Run("It should return an authorized response and send the expected request body", func(t *testing.T) {
		// Arrange
		var receivedBody AuthorizationRequest
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NoError(t, json.NewDecoder(r.Body).Decode(&receivedBody))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(AuthorizationResponse{Authorized: true, AuthorizationCode: "auth-code-123"})
		}))
		defer server.Close()

		client := NewClient(server.URL, time.Second)
		req := AuthorizationRequest{
			CardNumber: "2222405343248877",
			ExpiryDate: "04/2030",
			Currency:   "GBP",
			Amount:     100,
			Cvv:        "123",
		}

		// Act
		resp, err := client.Authorize(context.Background(), req)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, AuthorizationResponse{Authorized: true, AuthorizationCode: "auth-code-123"}, resp)
		assert.Equal(t, req, receivedBody)
	})

	t.Run("It should return a declined response when the bank declines the payment", func(t *testing.T) {
		// Arrange
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(AuthorizationResponse{Authorized: false})
		}))
		defer server.Close()

		client := NewClient(server.URL, time.Second)

		// Act
		resp, err := client.Authorize(context.Background(), AuthorizationRequest{})

		// Assert
		assert.NoError(t, err)
		assert.False(t, resp.Authorized)
	})

	t.Run("It should return ErrBankUnavailable when the bank returns a 503", func(t *testing.T) {
		// Arrange
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		client := NewClient(server.URL, time.Second)

		// Act
		_, err := client.Authorize(context.Background(), AuthorizationRequest{})

		// Assert
		assert.ErrorIs(t, err, ErrBankUnavailable)
	})

	t.Run("It should return ErrBankUnavailable when the bank returns a malformed body", func(t *testing.T) {
		// Arrange
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not json"))
		}))
		defer server.Close()

		client := NewClient(server.URL, time.Second)

		// Act
		_, err := client.Authorize(context.Background(), AuthorizationRequest{})

		// Assert
		assert.ErrorIs(t, err, ErrBankUnavailable)
	})

	t.Run("It should return ErrBankUnavailable when the bank does not respond within the timeout", func(t *testing.T) {
		// Arrange
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(50 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient(server.URL, 10*time.Millisecond)

		// Act
		_, err := client.Authorize(context.Background(), AuthorizationRequest{})

		// Assert
		assert.ErrorIs(t, err, ErrBankUnavailable)
	})
}
