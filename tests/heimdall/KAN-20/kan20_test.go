// KAN-20: Payment Refunds API — merchants refund payments in full or part.
// AC1: POST /payments/{id}/refunds returns 201 Created with refund body.
// AC2: POST /payments/{id}/refunds for missing payment returns 404.
// AC3: POST /payments/{id}/refunds with amount exceeding payment amount returns 422.
// Assumptions:
//   - Refund body has id, payment_id, amount, currency (same style as Payment).
//   - Error bodies follow existing errorJSON pattern: {"error": "..."}.
//   - X-Merchant-ID is required on the refund endpoint too.
//   - Minimum refund amount is 1, matching CreatePayment's minimum.
//   - Over-refunding is checked server-side: cumulative refunds ≤ payment amount.
package kan20_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aarush323/heimdall-fixture-pay/internal/app"
)

var fixedNow = time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)

func createPayment(t *testing.T, h http.Handler, merchant string) string {
	t.Helper()
	body := `{"amount": 2000, "currency": "USD"}`
	req := httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-ID", merchant)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create payment: status=%d body=%s", rec.Code, rec.Body)
	}
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || p.ID == "" {
		t.Fatalf("create payment: bad body: %v", err)
	}
	return p.ID
}

// heimdall:AC1
func TestRefundPayment_ReturnsCreatedRefund(t *testing.T) {
	h := app.New(app.Config{Now: func() time.Time { return fixedNow }})
	paymentID := createPayment(t, h, "m-1")

	body := `{"amount": 500}`
	req := httptest.NewRequest(http.MethodPost, "/payments/"+paymentID+"/refunds", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-ID", "m-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d want 201; body=%s", rec.Code, rec.Body)
	}
	var refund struct {
		ID        string `json:"id"`
		PaymentID string `json:"payment_id"`
		Amount    int64  `json:"amount"`
		Currency  string `json:"currency"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &refund); err != nil {
		t.Fatalf("response is not a refund: %v; body=%s", err, rec.Body)
	}
	if refund.ID == "" {
		t.Error("refund id is empty")
	}
	if refund.PaymentID != paymentID {
		t.Errorf("payment_id=%q want %q", refund.PaymentID, paymentID)
	}
	if refund.Amount != 500 {
		t.Errorf("amount=%d want 500", refund.Amount)
	}
	if refund.Currency != "USD" {
		t.Errorf("currency=%q want USD", refund.Currency)
	}
}

// heimdall:AC2
func TestRefundPayment_UnknownPayment_Returns404(t *testing.T) {
	h := app.New(app.Config{Now: func() time.Time { return fixedNow }})
	body := `{"amount": 500}`
	req := httptest.NewRequest(http.MethodPost, "/payments/does-not-exist/refunds", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-ID", "m-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404; body=%s", rec.Code, rec.Body)
	}
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("body is not error JSON: %v; body=%s", err, rec.Body)
	}
	if e.Error == "" {
		t.Error("error message is empty")
	}
}

// heimdall:AC3
func TestRefundPayment_AmountExceedsPayment_Returns422(t *testing.T) {
	h := app.New(app.Config{Now: func() time.Time { return fixedNow }})
	paymentID := createPayment(t, h, "m-1")

	body := `{"amount": 999999}`
	req := httptest.NewRequest(http.MethodPost, "/payments/"+paymentID+"/refunds", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-ID", "m-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want 422; body=%s", rec.Code, rec.Body)
	}
	var e struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("body is not error JSON: %v; body=%s", err, rec.Body)
	}
	if e.Error == "" {
		t.Error("error message is empty")
	}
}