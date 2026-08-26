package models

import "time"

const (
	minCardNumberLength = 14
	maxCardNumberLength = 19
	minCvvLength        = 3
	maxCvvLength        = 4
)

var allowedCurrencies = map[string]bool{
	"GBP": true,
	"USD": true,
	"EUR": true,
}

// Validate checks a payment request against the payment gateway's field
// rules and returns every violation found, rather than stopping at the
// first one, so a merchant can fix their integration in a single pass.
func (r PostPaymentRequest) Validate(now time.Time) []FieldError {
	var errs []FieldError

	if err := validateCardNumber(r.CardNumber); err != nil {
		errs = append(errs, *err)
	}

	if err := validateExpiryMonth(r.ExpiryMonth); err != nil {
		errs = append(errs, *err)
	} else if err := validateExpiryIsInTheFuture(r.ExpiryMonth, r.ExpiryYear, now); err != nil {
		errs = append(errs, *err)
	}

	if err := validateCurrency(r.Currency); err != nil {
		errs = append(errs, *err)
	}

	if err := validateAmount(r.Amount); err != nil {
		errs = append(errs, *err)
	}

	if err := validateCvv(r.Cvv); err != nil {
		errs = append(errs, *err)
	}

	return errs
}

func validateCardNumber(cardNumber string) *FieldError {
	switch {
	case cardNumber == "":
		return &FieldError{Field: "card_number", Message: "is required"}
	case len(cardNumber) < minCardNumberLength || len(cardNumber) > maxCardNumberLength:
		return &FieldError{Field: "card_number", Message: "must be between 14 and 19 characters long"}
	case !isDigitsOnly(cardNumber):
		return &FieldError{Field: "card_number", Message: "must only contain numeric characters"}
	}
	return nil
}

func validateExpiryMonth(month int) *FieldError {
	if month < 1 || month > 12 {
		return &FieldError{Field: "expiry_month", Message: "must be between 1 and 12"}
	}
	return nil
}

func validateExpiryIsInTheFuture(month, year int, now time.Time) *FieldError {
	expiry := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	if !expiry.After(currentMonth) {
		return &FieldError{Field: "expiry_year", Message: "must be in the future"}
	}
	return nil
}

func validateCurrency(currency string) *FieldError {
	switch {
	case currency == "":
		return &FieldError{Field: "currency", Message: "is required"}
	case len(currency) != 3:
		return &FieldError{Field: "currency", Message: "must be 3 characters"}
	case !allowedCurrencies[currency]:
		return &FieldError{Field: "currency", Message: "must be one of GBP, USD, EUR"}
	}
	return nil
}

func validateAmount(amount int) *FieldError {
	if amount <= 0 {
		return &FieldError{Field: "amount", Message: "must be a positive integer"}
	}
	return nil
}

func validateCvv(cvv string) *FieldError {
	switch {
	case cvv == "":
		return &FieldError{Field: "cvv", Message: "is required"}
	case len(cvv) < minCvvLength || len(cvv) > maxCvvLength:
		return &FieldError{Field: "cvv", Message: "must be 3 to 4 characters long"}
	case !isDigitsOnly(cvv):
		return &FieldError{Field: "cvv", Message: "must only contain numeric characters"}
	}
	return nil
}

func isDigitsOnly(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
