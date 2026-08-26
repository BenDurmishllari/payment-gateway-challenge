package repository

import (
	"sync"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
)

type PaymentsRepository struct {
	mu       sync.RWMutex
	payments map[string]models.GetPaymentResponse
}

func NewPaymentsRepository() *PaymentsRepository {
	return &PaymentsRepository{
		payments: make(map[string]models.GetPaymentResponse),
	}
}

func (ps *PaymentsRepository) GetPayment(id string) (models.GetPaymentResponse, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	payment, ok := ps.payments[id]
	return payment, ok
}

func (ps *PaymentsRepository) AddPayment(payment models.GetPaymentResponse) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.payments[payment.ID] = payment
}
