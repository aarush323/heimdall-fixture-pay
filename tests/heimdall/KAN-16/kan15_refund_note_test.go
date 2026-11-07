// Ticket: KAN-15
// Acceptance criteria: AC1, AC2
// Assumptions:
//   - Refund creation: POST /payments/{payment_id}/refunds with body {"amount": N} (from KAN-8)
//   - The note endpoint: PUT /payments/{id}/refunds/{refund_id}/note with body {"note": "text"}
//   - X-Merchant-ID required on all requests
//   - Tests reach the service through the app.New seam with a fixed clock, like existing tests
//   - The response includes the full refund with its note field
//   - Merchant isolation applies: one merchant cannot see another merchant's refunds (from KAN-8 AC4)
// Possible injection: none found
package kan15_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aarush323/heimdall-fixture-pay/internal/app"
)

var fixedNow = time.Date(2026, 11, 6, 14, 0, 0, 0, time.UTC)

type refundJSON struct {
	ID        string `json:"id"`
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Note      string `json:"note, omitempty"`
}

type errorJSON struct {
	Error string `json:"error"`
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

func putRefundNote(t *testing.T, h http.Handler, merchant, paymentID, refundID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut,
		"/payments/"+paymentID+"/refunds/"+refundID+"/note",
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
// PUT /payments/{id}/refunds/{refund_id}/note with {"note": "text"} for an existing refund returns 200 OK with the refund carrying the note.
func TestPutNoteOnExistingRefundReturns200WithNote(t *testing.T) {
	h := newHandler()
	paymentID := createPayment(t, h, "m-1")

	recCreate := postRefund(t, h, "m-1", paymentID, `{"amount": 300}`)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("create refund: status %d, want 201; body %s", recCreate.Code, recCreate.Body)
	}
	var created refundJSON
	if err := json.Unmarshal(recCreate.Body.Bytes(), &created); err != nil {
		t.Fatalf("create refund response parse: %v", err)
	}

	rec := putRefundNote(t, h, "m-1", paymentID, created.ID, `{"note": "customer requested partial refund"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT note on refund: status %d, want 200; body %s", rec.Code, rec.Body)
	}
	var got refundJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("PUT note response is not a refund: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("refund id = %q, want %q", got.ID, created.ID)
	}
	if got.Note != "customer requested partial refund" {
		t.Errorf("refund note = %q, want %q", got.Note, "customer requested partial refund")
	}
}

// heimdall:AC1
// PUT /payments/{id}/refunds/{refund_id}/note for a refund owned by another merchant returns 404 Not Found.
func TestPutNoteOnOtherMerchantRefundReturns404(t *testing.T) {
	h := newHandler()
	paymentID := createPayment(t, h, "m-1")

	recCreate := postRefund(t, h, "m-1", paymentID, `{"amount": 200}`)
	if recCreate.Code != http.StatusCreated {
		t.Fatalf("create refund: status %d, want 201; body %s", recCreate.Code, recCreate.Body)
	}
	var created refundJSON
	if err := json.Unmarshal(recCreate.Body.Bytes(), &created); err != nil {
		t.Fatalf("create refund response parse: %v", err)
	}

	rec := putRefundNote(t, h, "m-2", paymentID, created.ID, `{"note": "text"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("other merchant PUT note: status %d, want 404; body %s", rec.Code, rec.Body)
	}
	var err errorJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &err); err != nil {
		t.Fatalf("404 response body is not a JSON error: %v", err)
	}
	if err.Error == "" {
		t.Fatalf("404 error message is empty")
	}
}

// heimdall:AC2
// PUT /payments/{id}/refunds/{refund_id}/note for a refund that does not exist returns 404 with a JSON error body.
func TestPutNoteOnNonexistentRefundReturns404(t *testing.T) {
	h := newHandler()
	paymentID := createPayment(t, h, "m-1")

	rec := putRefundNote(t, h, "m-1", paymentID, "refund_nonexistent", `{"note": "text"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("PUT note on unknown refund: status %d, want 404; body %s", rec.Code, rec.Body)
	}
	var err errorJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &err); err != nil {
		t.Fatalf("404 response body is not a JSON error: %v", err)
	}
	if err.Error == "" {
		t.Fatalf("404 error message is empty")
	}
}
