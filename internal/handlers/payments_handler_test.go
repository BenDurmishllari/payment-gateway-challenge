package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/bank"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixtures struct {
	storage *repository.PaymentsRepository
	router  *chi.Mux
}

func newFixtures(t *testing.T, authorizer bank.Authorizer) fixtures {
	storage := repository.NewPaymentsRepository()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewPaymentsHandler(storage, authorizer, logger)

	router := chi.NewRouter()
	router.Get("/api/payments/{id}", handler.GetHandler())
	router.Post("/api/payments", handler.PostHandler())

	return fixtures{
		storage: storage,
		router:  router,
	}
}

// stubAuthorizer is a hand-rolled test double for bank.Authorizer. A single
// interface with one method doesn't justify pulling in gomock/mockgen.
type stubAuthorizer struct {
	authorize func(ctx context.Context, req bank.AuthorizationRequest) (bank.AuthorizationResponse, error)
	called    bool
}

func (s *stubAuthorizer) Authorize(ctx context.Context, req bank.AuthorizationRequest) (bank.AuthorizationResponse, error) {
	s.called = true
	return s.authorize(ctx, req)
}

func neverCalledAuthorizer(t *testing.T) *stubAuthorizer {
	return &stubAuthorizer{
		authorize: func(ctx context.Context, req bank.AuthorizationRequest) (bank.AuthorizationResponse, error) {
			t.Fatal("the bank should not have been called")
			return bank.AuthorizationResponse{}, nil
		},
	}
}

func aValidPaymentRequest() models.PostPaymentRequest {
	return models.PostPaymentRequest{
		CardNumber:  "2222405343248877",
		ExpiryMonth: 4,
		ExpiryYear:  2030,
		Currency:    "GBP",
		Amount:      100,
		Cvv:         "123",
	}
}

func postPayment(t *testing.T, router *chi.Mux, body any) *httptest.ResponseRecorder {
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func TestPaymentsHandler_GetHandler(t *testing.T) {
	t.Run("It should return the payment when it exists", func(t *testing.T) {
		// Arrange
		f := newFixtures(t, neverCalledAuthorizer(t))
		payment := models.GetPaymentResponse{
			ID:                 "test-id",
			PaymentStatus:      "irrelevant_status",
			CardNumberLastFour: "irrelevant_card_number_last_four",
			ExpiryMonth:        1,
			ExpiryYear:         2035,
			Currency:           "irrelevant_currency",
			Amount:             1,
		}
		f.storage.AddPayment(payment)

		// Act
		req := httptest.NewRequest(http.MethodGet, "/api/payments/test-id", nil)
		w := httptest.NewRecorder()
		f.router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("It should return 404 when the payment does not exist", func(t *testing.T) {
		// Arrange
		f := newFixtures(t, neverCalledAuthorizer(t))

		// Act
		req := httptest.NewRequest(http.MethodGet, "/api/payments/does-not-exist", nil)
		w := httptest.NewRecorder()
		f.router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestPaymentsHandler_PostHandler(t *testing.T) {
	t.Run("It should create an authorized payment when the bank authorizes it", func(t *testing.T) {
		// Arrange
		authorizer := &stubAuthorizer{
			authorize: func(ctx context.Context, req bank.AuthorizationRequest) (bank.AuthorizationResponse, error) {
				return bank.AuthorizationResponse{Authorized: true, AuthorizationCode: "irrelevant_authorization_code"}, nil
			},
		}
		f := newFixtures(t, authorizer)

		// Act
		w := postPayment(t, f.router, aValidPaymentRequest())

		// Assert
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp models.PostPaymentResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, models.StatusAuthorized, resp.PaymentStatus)
		assert.Equal(t, "8877", resp.CardNumberLastFour)
		assert.NotEmpty(t, resp.ID)
	})

	t.Run("It should create a declined payment when the bank declines it", func(t *testing.T) {
		// Arrange
		authorizer := &stubAuthorizer{
			authorize: func(ctx context.Context, req bank.AuthorizationRequest) (bank.AuthorizationResponse, error) {
				return bank.AuthorizationResponse{Authorized: false}, nil
			},
		}
		f := newFixtures(t, authorizer)

		// Act
		w := postPayment(t, f.router, aValidPaymentRequest())

		// Assert
		assert.Equal(t, http.StatusCreated, w.Code)

		var resp models.PostPaymentResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, models.StatusDeclined, resp.PaymentStatus)
	})

	t.Run("It should reject the payment without calling the bank when validation fails", func(t *testing.T) {
		// Arrange
		authorizer := neverCalledAuthorizer(t)
		f := newFixtures(t, authorizer)
		invalidRequest := aValidPaymentRequest()
		invalidRequest.Cvv = "12"

		// Act
		w := postPayment(t, f.router, invalidRequest)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.False(t, authorizer.called)

		var resp models.RejectedResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, models.StatusRejected, resp.PaymentStatus)
		assert.Equal(t, []models.FieldError{{Field: "cvv", Message: "must be 3 to 4 characters long"}}, resp.Errors)
	})

	t.Run("It should reject the payment when the request body is not valid JSON", func(t *testing.T) {
		// Arrange
		authorizer := neverCalledAuthorizer(t)
		f := newFixtures(t, authorizer)

		// Act
		req := httptest.NewRequest(http.MethodPost, "/api/payments", bytes.NewReader([]byte("not json")))
		w := httptest.NewRecorder()
		f.router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.False(t, authorizer.called)
	})

	t.Run("It should return a bad gateway response when the bank is unavailable", func(t *testing.T) {
		// Arrange
		authorizer := &stubAuthorizer{
			authorize: func(ctx context.Context, req bank.AuthorizationRequest) (bank.AuthorizationResponse, error) {
				return bank.AuthorizationResponse{}, bank.ErrBankUnavailable
			},
		}
		f := newFixtures(t, authorizer)

		// Act
		w := postPayment(t, f.router, aValidPaymentRequest())

		// Assert
		assert.Equal(t, http.StatusBadGateway, w.Code)
	})

	t.Run("It should make the created payment retrievable by id", func(t *testing.T) {
		// Arrange
		authorizer := &stubAuthorizer{
			authorize: func(ctx context.Context, req bank.AuthorizationRequest) (bank.AuthorizationResponse, error) {
				return bank.AuthorizationResponse{Authorized: true, AuthorizationCode: "irrelevant_authorization_code"}, nil
			},
		}
		f := newFixtures(t, authorizer)
		created := postPayment(t, f.router, aValidPaymentRequest())
		var createdPayment models.PostPaymentResponse
		require.NoError(t, json.Unmarshal(created.Body.Bytes(), &createdPayment))

		// Act
		req := httptest.NewRequest(http.MethodGet, "/api/payments/"+createdPayment.ID, nil)
		w := httptest.NewRecorder()
		f.router.ServeHTTP(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var fetchedPayment models.GetPaymentResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &fetchedPayment))
		assert.Equal(t, createdPayment.ID, fetchedPayment.ID)
		assert.Equal(t, models.StatusAuthorized, fetchedPayment.PaymentStatus)
	})
}
