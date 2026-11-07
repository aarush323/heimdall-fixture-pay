package payments

import (
	"errors"
	"fmt"
	"time"
)

// ErrPaymentNotFound means no payment has the requested id.
var ErrPaymentNotFound = errors.New("payment not found")

// ErrRefundTooLarge means the refund asks for more than the payment allows.
var ErrRefundTooLarge = errors.New("refund amount is above the payment amount")

// ErrInvalidRefund means the refund amount is not a positive number of the smallest currency unit.
var ErrInvalidRefund = errors.New("refund amount is below 1")

// Refund returns money from a payment to the customer. Amount is in the smallest unit of Currency.
type Refund struct {
	ID        string
	PaymentID string
	Amount    int64
	Currency  string
	CreatedAt time.Time
}

// Refund records a refund of amount on the payment with paymentID and returns it.
func (s *Service) Refund(paymentID string, amount int64) (Refund, error) {
	p, ok := s.store.Get(paymentID)
	if !ok {
		return Refund{}, fmt.Errorf("refund %s: %w", paymentID, ErrPaymentNotFound)
	}
	if amount < 1 {
		return Refund{}, fmt.Errorf("refund %s: %w", paymentID, ErrInvalidRefund)
	}
	if amount > p.Amount {
		return Refund{}, fmt.Errorf("refund %s: %w", paymentID, ErrRefundTooLarge)
	}
	r := Refund{
		ID:        s.newID(),
		PaymentID: p.ID,
		Amount:    amount,
		Currency:  p.Currency,
		CreatedAt: s.now().UTC(),
	}
	s.store.PutRefund(r)
	return r, nil
}

// RefundCount returns how many refunds the payment with paymentID has.
func (s *Service) RefundCount(paymentID string) (int, error) {
	if _, ok := s.store.Get(paymentID); !ok {
		return 0, fmt.Errorf("refund count %s: %w", paymentID, ErrPaymentNotFound)
	}
	return s.store.CountRefunds(paymentID), nil
}
