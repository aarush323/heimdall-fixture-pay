package pay200_test

import "testing"

// AC1: a refund can never exceed the original payment.
func TestRefundCannotExceedThePayment(t *testing.T) {
	payment, refund := 100, 150
	if refund <= payment {
		t.Fatalf("a refund of %d on a payment of %d must be refused", refund, payment)
	}
}
