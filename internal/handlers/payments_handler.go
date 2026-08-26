package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/bank"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PaymentsHandler struct {
	storage    *repository.PaymentsRepository
	authorizer bank.Authorizer
}

func NewPaymentsHandler(storage *repository.PaymentsRepository, authorizer bank.Authorizer) *PaymentsHandler {
	return &PaymentsHandler{
		storage:    storage,
		authorizer: authorizer,
	}
}

// GetHandler returns an http.HandlerFunc that handles HTTP GET requests.
// It retrieves a payment record by its ID from the storage.
// The ID is expected to be part of the URL.
func (h *PaymentsHandler) GetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		payment, ok := h.storage.GetPayment(id)

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(payment); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// PostHandler returns an http.HandlerFunc that handles HTTP POST requests.
// It validates the payment request, forwards it to the acquiring bank, and
// stores the outcome. A validation failure rejects the request without
// calling the bank; a bank failure is reported without storing a payment,
// since no definite outcome was received.
func (h *PaymentsHandler) PostHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.PostPaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeRejected(w, []models.FieldError{{Field: "body", Message: "must be valid JSON"}})
			return
		}

		if errs := req.Validate(time.Now()); len(errs) > 0 {
			writeRejected(w, errs)
			return
		}

		authResp, err := h.authorizer.Authorize(r.Context(), toAuthorizationRequest(req))
		if err != nil {
			w.WriteHeader(bankErrorStatusCode(err))
			return
		}

		status := models.StatusDeclined
		if authResp.Authorized {
			status = models.StatusAuthorized
		}

		id := uuid.NewString()
		postResponse, getResponse := buildPaymentResponses(id, status, req)

		h.storage.AddPayment(getResponse)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(postResponse)
	}
}

func writeRejected(w http.ResponseWriter, errs []models.FieldError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(models.RejectedResponse{
		PaymentStatus: models.StatusRejected,
		Errors:        errs,
	})
}

// toAuthorizationRequest builds the bank's request shape from a payment
// request, formatting the expiry date as the zero-padded MM/YYYY the bank
// simulator expects.
func toAuthorizationRequest(req models.PostPaymentRequest) bank.AuthorizationRequest {
	return bank.AuthorizationRequest{
		CardNumber: req.CardNumber,
		ExpiryDate: fmt.Sprintf("%02d/%d", req.ExpiryMonth, req.ExpiryYear),
		Currency:   req.Currency,
		Amount:     req.Amount,
		Cvv:        req.Cvv,
	}
}

// bankErrorStatusCode maps a bank call failure to the HTTP status returned
// to the merchant.
func bankErrorStatusCode(err error) int {
	if errors.Is(err, bank.ErrBankUnavailable) {
		return http.StatusBadGateway
	}
	return http.StatusInternalServerError
}

// buildPaymentResponses builds the reply sent back to the merchant and the
// record stored for later retrieval, from the same underlying values.
func buildPaymentResponses(id, status string, req models.PostPaymentRequest) (models.PostPaymentResponse, models.GetPaymentResponse) {
	lastFour := req.CardNumber[len(req.CardNumber)-4:]

	postResponse := models.PostPaymentResponse{
		ID:                 id,
		PaymentStatus:      status,
		CardNumberLastFour: lastFour,
		ExpiryMonth:        req.ExpiryMonth,
		ExpiryYear:         req.ExpiryYear,
		Currency:           req.Currency,
		Amount:             req.Amount,
	}

	getResponse := models.GetPaymentResponse{
		ID:                 postResponse.ID,
		PaymentStatus:      postResponse.PaymentStatus,
		CardNumberLastFour: postResponse.CardNumberLastFour,
		ExpiryMonth:        postResponse.ExpiryMonth,
		ExpiryYear:         postResponse.ExpiryYear,
		Currency:           postResponse.Currency,
		Amount:             postResponse.Amount,
	}

	return postResponse, getResponse
}
