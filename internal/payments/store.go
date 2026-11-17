package payments

import "sync"

// MemoryStore keeps payments in memory. It is safe for concurrent use.
type MemoryStore struct {
	mu       sync.Mutex
	payments map[string]Payment
	order    []string // ids in the order they were first recorded
}

// NewMemoryStore returns an empty store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{payments: make(map[string]Payment)}
}

// Put records p under its ID, replacing any payment with that ID.
func (s *MemoryStore) Put(p Payment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, seen := s.payments[p.ID]; !seen {
		s.order = append(s.order, p.ID)
	}
	s.payments[p.ID] = p
}

// ByMerchant returns merchantID's payments, newest recorded first. Another merchant's payments never appear.
func (s *MemoryStore) ByMerchant(merchantID string) []Payment {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Payment{}
	for i := len(s.order) - 1; i >= 0; i-- {
		if p := s.payments[s.order[i]]; p.MerchantID == merchantID {
			out = append(out, p)
		}
	}
	return out
}

// Get returns the payment with id and whether it exists.
func (s *MemoryStore) Get(id string) (Payment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.payments[id]
	return p, ok
}
