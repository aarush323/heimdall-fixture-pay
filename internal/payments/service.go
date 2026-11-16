package payments

import (
	"errors"
	"fmt"
	"time"
)

// ErrNotFound is a payment that does not exist for the caller: unknown, or another merchant's. The two are one answer,
// so a caller cannot learn that another merchant's payment exists.
var ErrNotFound = errors.New("payment not found")

// Service creates payments. Now and newID are injected, so tests need no real clock or randomness.
type Service struct {
	store *MemoryStore
	now   func() time.Time
	newID func() string
}

// NewService returns a Service that records payments in store.
func NewService(store *MemoryStore, now func() time.Time, newID func() string) *Service {
	return &Service{store: store, now: now, newID: newID}
}

// Create validates the request, records the payment for merchantID and returns it.
func (s *Service) Create(merchantID string, amount int64, currency string) (Payment, error) {
	if err := ValidateNew(merchantID, amount, currency); err != nil {
		return Payment{}, fmt.Errorf("create payment: %w", err)
	}
	p := Payment{
		ID:         s.newID(),
		MerchantID: merchantID,
		Amount:     amount,
		Currency:   currency,
		CreatedAt:  s.now().UTC(),
	}
	s.store.Put(p)
	return p, nil
}

// Get returns merchantID's payment with id. Another merchant's payment is ErrNotFound, the same as an unknown id.
func (s *Service) Get(merchantID, id string) (Payment, error) {
	p, ok := s.store.Get(id)
	if !ok || p.MerchantID != merchantID {
		return Payment{}, ErrNotFound
	}
	return p, nil
}
