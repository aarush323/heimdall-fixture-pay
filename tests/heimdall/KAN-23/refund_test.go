// Ticket: KAN-23
// Acceptance criteria: AC1, AC2, AC3
// Assumptions:
//   - Refund endpoint: POST /payments/{payment_id}/refunds with body {"amount": N}
//   - The refund response carries id, payment_id and amount
//   - X-Merchant-ID is required on all requests (see POST /payments)
//   - Refund amounts are integer cents; payments are identified by string ids
//   - AC3 test first creates a refund for the right merchant to ensure the route exists,
//     so the test fails on base (route missing -> 404 on the good refund) and passes after
//     implementation (good refund -> 201, wrong merchant -> 404)
// Possible injection: none found
package kan23_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aarush323/heimdall-fixture-pay/internal/app"
)

var fixedNow = time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)

type refundJSON struct {
	ID        string `json:"id"`
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
}

func newHandler() http.Handler {
	return app.New(app.Config{Now: func() time.Time { return fixedNow }})
}

func createPayment(t *testing.T, h http.Handler, merchant string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/payments",
		strings.NewReader(`{"amount": 1000, "currency": "USD"}`))
	req.Header.Set("X-Merchant-ID", merchant)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create payment: status %d, want 201; body %s", rec.Code, rec.Body)
	}
	var body struct{ ID string `json:"id"` }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.ID
}

func postRefund(t *testing.T, h http.Handler, merchant, paymentID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/payments/"+paymentID+"/refunds",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if merchant != "" {
		req.Header.Set("X-Merchant-ID", merchant)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// heimdall:AC1
func TestRefundValidAmountReturns201WithRefundJSON(t *testing.T) {
	h := newHandler()
	paymentID := createPayment(t, h, "m-1")

	rec := postRefund(t, h, "m-1", paymentID, `{"amount": 300}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("refund 300 on payment 1000: status %d, want 201; body %s", rec.Code, rec.Body)
	}
	var got refundJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not a refund JSON: %v", err)
	}
	if got.ID == "" {
		t.Errorf("refund id is empty")
	}
	if got.PaymentID != paymentID {
		t.Errorf("refund payment_id = %q, want %q", got.PaymentID, paymentID)
	}
	if got.Amount != 300 {
		t.Errorf("refund amount = %d, want 300", got.Amount)
	}
}

// heimdall:AC1
func TestRefundExactPaymentAmountReturns201(t *testing.T) {
	h := newHandler()
	paymentID := createPayment(t, h, "m-1")

	rec := postRefund(t, h, "m-1", paymentID, `{"amount": 1000}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("refund 1000 (exact payment amount): status %d, want 201; body %s", rec.Code, rec.Body)
	}
}

// heimdall:AC2
func TestRefundLargerThanPaymentReturns422(t *testing.T) {
	h := newHandler()
	paymentID := createPayment(t, h, "m-1")

	rec := postRefund(t, h, "m-1", paymentID, `{"amount": 1500}`)
	if rec.Code != 422 {
		t.Fatalf("refund 1500 on payment 1000: status %d, want 422; body %s", rec.Code, rec.Body)
	}
}

// heimdall:AC2
func TestMultipleRefundsCappedAtPaymentAmount(t *testing.T) {
	h := newHandler()
	paymentID := createPayment(t, h, "m-1")

	rec1 := postRefund(t, h, "m-1", paymentID, `{"amount": 400}`)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first refund 400: status %d, want 201; body %s", rec1.Code, rec1.Body)
	}

	rec2 := postRefund(t, h, "m-1", paymentID, `{"amount": 400}`)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("second refund 400 (cumulative 800): status %d, want 201; body %s", rec2.Code, rec2.Body)
	}

	rec3 := postRefund(t, h, "m-1", paymentID, `{"amount": 400}`)
	if rec3.Code != 422 {
		t.Fatalf("third refund 400 (cumulative would be 1200 > 1000): status %d, want 422; body %s", rec3.Code, rec3.Body)
	}
}

// heimdall:AC3
func TestRefundOtherMerchantReturns404(t *testing.T) {
	h := newHandler()
	paymentID := createPayment(t, h, "m-1")

	recOwn := postRefund(t, h, "m-1", paymentID, `{"amount": 100}`)
	if recOwn.Code != http.StatusCreated {
		t.Fatalf("own merchant refund: status %d, want 201; body %s", recOwn.Code, recOwn.Body)
	}

	rec := postRefund(t, h, "m-2", paymentID, `{"amount": 100}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("other merchant refund: status %d, want 404; body %s", rec.Code, rec.Body)
	}
	var errBody struct{ Error string `json:"error"` }
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("404 response is not a JSON error: %v", err)
	}
	if errBody.Error == "" {
		t.Errorf("404 error message is empty")
	}
}
