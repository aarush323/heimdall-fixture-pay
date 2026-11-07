// KAN-17: refund receipt endpoint.
// AC1: GET /payments/{id}/refunds/{refund_id}/receipt for an existing refund returns 200 with receipt JSON.
// AC2: GET /payments/{id}/refunds/{refund_id}/receipt for a non-existent refund returns 404 with JSON error.
//
// Assumptions:
// - The receipt endpoint uses path /payments/{payment_id}/refunds/{refund_id}/receipt.
// - Refunds exist under a payment, seeded or created as part of the implementation.
// - The 404 for a missing refund carries Content-Type: application/json with {"error": "..."}.
// - No injection found in ticket text.
package kan17_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aarush323/heimdall-fixture-pay/internal/app"
)

func createPayment(t *testing.T, h http.Handler, merchant, body string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-ID", merchant)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create payment: status %d, body %s", rec.Code, rec.Body)
	}
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("create payment body: %v", err)
	}
	return p.ID
}

func get(t *testing.T, h http.Handler, url string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// heimdall:AC1
func TestRefundReceipt_ExistingRefund_Returns200(t *testing.T) {
	h := app.New(app.Config{})
	paymentID := createPayment(t, h, "m-1", `{"amount": 1000, "currency": "USD"}`)

	rec := get(t, h, "/payments/"+paymentID+"/refunds/ref_1/receipt")
	if rec.Code != http.StatusOK {
		t.Fatalf("receipt status = %d, want 200; body %s", rec.Code, rec.Body)
	}

	var receipt struct {
		RefundID string `json:"refund_id"`
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &receipt); err != nil {
		t.Fatalf("receipt body is not JSON: %v", err)
	}
	if receipt.RefundID == "" {
		t.Error("receipt.refund_id is empty")
	}
	if receipt.Amount != 1000 {
		t.Errorf("receipt.amount = %d, want 1000", receipt.Amount)
	}
	if receipt.Currency != "USD" {
		t.Errorf("receipt.currency = %q, want USD", receipt.Currency)
	}
}

// heimdall:AC2
func TestRefundReceipt_NonExistentRefund_Returns404(t *testing.T) {
	h := app.New(app.Config{})
	paymentID := createPayment(t, h, "m-1", `{"amount": 500, "currency": "EUR"}`)

	rec := get(t, h, "/payments/"+paymentID+"/refunds/no_such_refund/receipt")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("receipt status = %d, want 404; body %s", rec.Code, rec.Body)
	}

	// On base the default 404 handler sends text/plain.
	// The implementation must return application/json.
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var errBody struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatalf("404 body is not JSON: %v", err)
	}
	if errBody.Error == "" {
		t.Error("404 error body has empty message")
	}
}
