package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	logger     *slog.Logger
}

func NewPaymentsHandler(storage *repository.PaymentsRepository, authorizer bank.Authorizer, logger *slog.Logger) *PaymentsHandler {
	return &PaymentsHandler{
		storage:    storage,
		authorizer: authorizer,
		logger:     logger,
	}
}

// GetHandler retrieves a previously processed payment by id.
//
//	@Summary		Get a payment
//	@Description	Retrieves a previously processed payment by its id
//	@Tags			payments
//	@Produce		json
//	@Param			id	path		string	true	"Payment ID"
//	@Success		200	{object}	models.GetPaymentResponse
//	@Failure		404	{string}	string	"payment not found"
//	@Router			/api/payments/{id} [get]
func (h *PaymentsHandler) GetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		payment, ok := h.storage.GetPayment(id)

		if !ok {
			h.logger.InfoContext(r.Context(), "payment not found", slog.String("payment_id", id))
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(payment); err != nil {
			h.logger.ErrorContext(r.Context(), "failed to encode payment response", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// PostHandler validates the payment request, forwards it to the acquiring bank,
// and stores the outcome. Rejected payments (validation failures) are never
// forwarded to the bank and are not stored.
//
//	@Summary		Process a payment
//	@Description	Validates a card payment and forwards it to the acquiring bank
//	@Tags			payments
//	@Accept			json
//	@Produce		json
//	@Param			payment	body		models.PostPaymentRequest	true	"Payment details"
//	@Success		201		{object}	models.PostPaymentResponse
//	@Failure		400		{object}	models.RejectedResponse	"validation failed"
//	@Failure		502		{string}	string	"bank unavailable"
//	@Router			/api/payments [post]
func (h *PaymentsHandler) PostHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.PostPaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logger.WarnContext(r.Context(), "invalid payment request body", slog.Any("error", err))
			h.writeRejected(r.Context(), w, []models.FieldError{{Field: "body", Message: "must be valid JSON"}})
			return
		}

		if errs := req.Validate(time.Now()); len(errs) > 0 {
			h.logger.InfoContext(r.Context(), "payment rejected", slog.Int("error_count", len(errs)))
			h.writeRejected(r.Context(), w, errs)
			return
		}

		authResp, err := h.authorizer.Authorize(r.Context(), toAuthorizationRequest(req))
		if err != nil {
			h.logger.ErrorContext(r.Context(), "bank authorization failed", slog.Any("error", err))
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

		h.logger.InfoContext(r.Context(), "payment processed",
			slog.String("payment_id", id),
			slog.String("status", status),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(postResponse); err != nil {
			h.logger.ErrorContext(r.Context(), "failed to encode payment response", slog.Any("error", err))
		}
	}
}

func (h *PaymentsHandler) writeRejected(ctx context.Context, w http.ResponseWriter, errs []models.FieldError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	rejected := models.RejectedResponse{
		PaymentStatus: models.StatusRejected,
		Errors:        errs,
	}
	if err := json.NewEncoder(w).Encode(rejected); err != nil {
		h.logger.ErrorContext(ctx, "failed to encode rejected-payment response", slog.Any("error", err))
	}
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
