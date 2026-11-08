package payments

import (
	"errors"
	"fmt"
	"sync"
)

// ErrNotFound says the payment does not exist for the calling merchant. A payment of another merchant is reported the
// same way, so ids cannot be probed.
var ErrNotFound = errors.New("payment not found")

// ErrExceedsPayment says a refund would take the refunds of a payment past the payment amount.
var ErrExceedsPayment = errors.New("refund exceeds the payment amount")

// Refund is money given back on part or all of a payment. Amount is in the smallest unit of the payment's currency.
type Refund struct {
	ID        string
	PaymentID string
	Amount    int64
}

// refundLog keeps refunds in memory. It is safe for concurrent use.
type refundLog struct {
	mu      sync.Mutex
	refunds []Refund
}

func (l *refundLog) add(r Refund) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refunds = append(l.refunds, r)
}

// total returns the sum of the refunds already made on paymentID.
func (l *refundLog) total(paymentID string) int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	var sum int64
	for _, r := range l.refunds {
		if r.PaymentID == paymentID {
			sum += r.Amount
		}
	}
	return sum
}

// Refund gives back amount on the payment paymentID of merchantID.
func (s *Service) Refund(merchantID, paymentID string, amount int64) (Refund, error) {
	if amount < 1 {
		return Refund{}, fmt.Errorf("refund: amount %d is below 1", amount)
	}
	p, ok := s.store.Get(paymentID)
	if !ok || p.MerchantID != merchantID {
		return Refund{}, ErrNotFound
	}
	if s.refunds.total(p.ID)+amount > p.Amount {
		return Refund{}, ErrExceedsPayment
	}
	r := Refund{ID: s.newID(), PaymentID: p.ID, Amount: amount}
	s.refunds.add(r)
	return r, nil
}
