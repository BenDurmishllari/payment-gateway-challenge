package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func aValidPaymentRequest() PostPaymentRequest {
	return PostPaymentRequest{
		CardNumber:  "2222405343248877",
		ExpiryMonth: 4,
		ExpiryYear:  2030,
		Currency:    "GBP",
		Amount:      100,
		Cvv:         "123",
	}
}

func TestPostPaymentRequest_Validate(t *testing.T) {
	now := time.Date(2024, time.June, 15, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		request        PostPaymentRequest
		expectedErrors []FieldError
	}{
		"It should accept a fully valid payment request": {
			request:        aValidPaymentRequest(),
			expectedErrors: nil,
		},
		"It should reject a payment when the card number is missing": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.CardNumber = ""
				return r
			}(),
			expectedErrors: []FieldError{{Field: "card_number", Message: "is required"}},
		},
		"It should reject a payment when the card number is too short": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.CardNumber = "1234567890123"
				return r
			}(),
			expectedErrors: []FieldError{{Field: "card_number", Message: "must be between 14 and 19 characters long"}},
		},
		"It should reject a payment when the card number is too long": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.CardNumber = "12345678901234567890"
				return r
			}(),
			expectedErrors: []FieldError{{Field: "card_number", Message: "must be between 14 and 19 characters long"}},
		},
		"It should reject a payment when the card number contains non-numeric characters": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.CardNumber = "2222abcd43248877"
				return r
			}(),
			expectedErrors: []FieldError{{Field: "card_number", Message: "must only contain numeric characters"}},
		},
		"It should reject a payment when the expiry month is zero": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.ExpiryMonth = 0
				return r
			}(),
			expectedErrors: []FieldError{{Field: "expiry_month", Message: "must be between 1 and 12"}},
		},
		"It should reject a payment when the expiry month is above 12": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.ExpiryMonth = 13
				return r
			}(),
			expectedErrors: []FieldError{{Field: "expiry_month", Message: "must be between 1 and 12"}},
		},
		"It should reject a payment when the expiry date is in the past": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.ExpiryMonth = 5
				r.ExpiryYear = 2024
				return r
			}(),
			expectedErrors: []FieldError{{Field: "expiry_year", Message: "must be in the future"}},
		},
		"It should reject a payment when the expiry date is the current month": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.ExpiryMonth = 6
				r.ExpiryYear = 2024
				return r
			}(),
			expectedErrors: []FieldError{{Field: "expiry_year", Message: "must be in the future"}},
		},
		"It should reject a payment when the currency is missing": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.Currency = ""
				return r
			}(),
			expectedErrors: []FieldError{{Field: "currency", Message: "is required"}},
		},
		"It should reject a payment when the currency is not 3 characters": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.Currency = "GB"
				return r
			}(),
			expectedErrors: []FieldError{{Field: "currency", Message: "must be 3 characters"}},
		},
		"It should reject a payment when the currency is not supported": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.Currency = "JPY"
				return r
			}(),
			expectedErrors: []FieldError{{Field: "currency", Message: "must be one of GBP, USD, EUR"}},
		},
		"It should reject a payment when the amount is zero": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.Amount = 0
				return r
			}(),
			expectedErrors: []FieldError{{Field: "amount", Message: "must be a positive integer"}},
		},
		"It should reject a payment when the amount is negative": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.Amount = -100
				return r
			}(),
			expectedErrors: []FieldError{{Field: "amount", Message: "must be a positive integer"}},
		},
		"It should reject a payment when the cvv is missing": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.Cvv = ""
				return r
			}(),
			expectedErrors: []FieldError{{Field: "cvv", Message: "is required"}},
		},
		"It should reject a payment when the cvv is too short": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.Cvv = "12"
				return r
			}(),
			expectedErrors: []FieldError{{Field: "cvv", Message: "must be 3 to 4 characters long"}},
		},
		"It should reject a payment when the cvv is too long": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.Cvv = "12345"
				return r
			}(),
			expectedErrors: []FieldError{{Field: "cvv", Message: "must be 3 to 4 characters long"}},
		},
		"It should reject a payment when the cvv contains non-numeric characters": {
			request: func() PostPaymentRequest {
				r := aValidPaymentRequest()
				r.Cvv = "12a"
				return r
			}(),
			expectedErrors: []FieldError{{Field: "cvv", Message: "must only contain numeric characters"}},
		},
		"It should return every field error when all fields are invalid": {
			request: PostPaymentRequest{
				CardNumber:  "",
				ExpiryMonth: 13,
				ExpiryYear:  2024,
				Currency:    "",
				Amount:      0,
				Cvv:         "",
			},
			expectedErrors: []FieldError{
				{Field: "card_number", Message: "is required"},
				{Field: "expiry_month", Message: "must be between 1 and 12"},
				{Field: "currency", Message: "is required"},
				{Field: "amount", Message: "must be a positive integer"},
				{Field: "cvv", Message: "is required"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			request := test.request

			// Act
			errs := request.Validate(now)

			// Assert
			assert.Equal(t, test.expectedErrors, errs)
		})
	}
}
