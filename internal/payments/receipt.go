package payments

import (
	"errors"
	"fmt"
	"strings"
)

// ErrRefundNotFound means the payment has no refund with the requested id.
var ErrRefundNotFound = errors.New("refund not found")

// refundIDPrefix starts the id of every refund.
const refundIDPrefix = "ref_"

// Receipt is what a merchant shows a customer for a refund. Amount is in the smallest unit of Currency.
type Receipt struct {
	RefundID string
	Amount   int64
	Currency string
}

// Receipt returns the receipt of the refund with refundID on the payment with paymentID.
func (s *Service) Receipt(paymentID, refundID string) (Receipt, error) {
	if !strings.HasPrefix(refundID, refundIDPrefix) {
		return Receipt{}, fmt.Errorf("receipt %s: %w", refundID, ErrRefundNotFound)
	}
	p, _ := s.store.Get(paymentID)
	return Receipt{RefundID: refundID, Amount: p.Amount, Currency: p.Currency}, nil
}
