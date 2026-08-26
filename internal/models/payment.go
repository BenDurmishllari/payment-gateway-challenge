package models

const (
	StatusAuthorized = "Authorized"
	StatusDeclined   = "Declined"
	StatusRejected   = "Rejected"
)

type PostPaymentRequest struct {
	CardNumber  string `json:"card_number"`
	ExpiryMonth int    `json:"expiry_month"`
	ExpiryYear  int    `json:"expiry_year"`
	Currency    string `json:"currency"`
	Amount      int    `json:"amount"`
	Cvv         string `json:"cvv"`
}

type PostPaymentResponse struct {
	Id                 string `json:"id"`
	PaymentStatus      string `json:"payment_status"`
	CardNumberLastFour string `json:"card_number_last_four"`
	ExpiryMonth        int    `json:"expiry_month"`
	ExpiryYear         int    `json:"expiry_year"`
	Currency           string `json:"currency"`
	Amount             int    `json:"amount"`
}

type GetPaymentResponse struct {
	Id                 string `json:"id"`
	PaymentStatus      string `json:"payment_status"`
	CardNumberLastFour string `json:"card_number_last_four"`
	ExpiryMonth        int    `json:"expiry_month"`
	ExpiryYear         int    `json:"expiry_year"`
	Currency           string `json:"currency"`
	Amount             int    `json:"amount"`
}

// FieldError describes a single validation failure on a request field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// RejectedResponse is returned when a payment request fails validation
// before the acquiring bank is ever called.
type RejectedResponse struct {
	PaymentStatus string       `json:"payment_status"`
	Errors        []FieldError `json:"errors"`
}
