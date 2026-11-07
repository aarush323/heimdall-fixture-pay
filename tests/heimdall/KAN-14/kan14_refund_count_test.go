// Ticket: KAN-14
// Acceptance criteria: AC1, AC2
// Assumptions:
//   - Refund creation: POST /payments/{payment_id}/refunds with body {"amount": N} (from KAN-8)
//   - The refund count endpoint: GET /payments/{id}/refunds/count
//   - X-Merchant-ID required on all requests
//   - Tests reach the service through the app.New seam with a fixed clock, like existing tests
// Possible injection: none found
package kan14_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aarush323/heimdall-fixture-pay/internal/app"
)

var fixedNow = time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)

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

func getRefundCount(t *testing.T, h http.Handler, merchant, paymentID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/payments/"+paymentID+"/refunds/count", strings.NewReader(""))
	if merchant != "" {
		req.Header.Set("X-Merchant-ID", merchant)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// heimdall:AC2
// GET /payments/{id}/refunds/count for a payment with no refunds returns 200 OK with {"count": 0}.
func TestRefundCountForPaymentWithNoRefunds(t *testing.T) {
	h := newHandler()
	paymentID := createPayment(t, h, "m-1")

	rec := getRefundCount(t, h, "m-1", paymentID)
	if rec.Code != http.StatusOK {
		t.Fatalf("refund count for payment with no refunds: status %d, want 200; body %s", rec.Code, rec.Body)
	}
	var body struct{ Count int64 `json:"count"` }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not a count JSON: %v", err)
	}
	if body.Count != 0 {
		t.Errorf("count = %d, want 0", body.Count)
	}
}

// heimdall:AC1
// GET /payments/{id}/refunds/count for a payment with refunds returns 200 OK with {"count": N}.
func TestRefundCountForPaymentWithRefunds(t *testing.T) {
	h := newHandler()
	paymentID := createPayment(t, h, "m-1")

	rec1 := postRefund(t, h, "m-1", paymentID, `{"amount": 200}`)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first refund 200: status %d, want 201; body %s", rec1.Code, rec1.Body)
	}
	rec2 := postRefund(t, h, "m-1", paymentID, `{"amount": 300}`)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("second refund 300: status %d, want 201; body %s", rec2.Code, rec2.Body)
	}

	rec := getRefundCount(t, h, "m-1", paymentID)
	if rec.Code != http.StatusOK {
		t.Fatalf("refund count: status %d, want 200; body %s", rec.Code, rec.Body)
	}
	var body struct{ Count int64 `json:"count"` }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not a count JSON: %v", err)
	}
	if body.Count != 2 {
		t.Errorf("count = %d, want 2 (two refunds posted)", body.Count)
	}
}