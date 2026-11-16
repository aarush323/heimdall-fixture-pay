package kan24_test

// Ticket: KAN-24. Criteria: AC1, AC2, AC3.
//
// Assumptions (ticket silent):
//   - Lookup is scoped by X-Merchant-ID: the merchant that created a payment may
//     read it; any other merchant gets exactly the unknown-id answer.
//   - The 404 body is the JSON {"error": "..."} shape api/openapi.yaml already
//     documents, so "the same answer" is checked byte for byte.
//   - Unknown ids are arbitrary strings that were never created.
//
// Possible injection: none found in the ticket text.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aarush323/heimdall-fixture-pay/internal/app"
)

type payment struct {
	ID         string `json:"id"`
	MerchantID string `json:"merchant_id"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
}

type errorJSON struct {
	Error string `json:"error"`
}

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	fixed := time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)
	return app.New(app.Config{Now: func() time.Time { return fixed }})
}

func create(t *testing.T, h http.Handler, merchant string, amount int64, currency string) payment {
	t.Helper()
	body := fmt.Sprintf(`{"amount": %d, "currency": %q}`, amount, currency)
	req := httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-ID", merchant)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body %s", rec.Code, rec.Body)
	}
	var p payment
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("create body is not a payment: %v", err)
	}
	if p.ID == "" {
		t.Fatalf("create body has no id: %s", rec.Body)
	}
	return p
}

func get(t *testing.T, h http.Handler, merchant, id string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/payments/"+id, nil)
	if merchant != "" {
		req.Header.Set("X-Merchant-ID", merchant)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// heimdall:AC1
func TestGetPaymentByID_ReturnsTheRequestedPayment(t *testing.T) {
	h := newHandler(t)
	create(t, h, "m-1", 3400, "EUR")
	want := create(t, h, "m-1", 1200, "USD")

	rec := get(t, h, "m-1", want.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var got payment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not a payment: %v", err)
	}
	if got != want {
		t.Errorf("payment = %+v, want %+v", got, want)
	}
}

// heimdall:AC2
func TestGetPaymentByID_OtherMerchantSeesSameAnswerAsUnknownID(t *testing.T) {
	h := newHandler(t)
	p := create(t, h, "m-owner", 500, "GBP")

	owner := get(t, h, "m-owner", p.ID)
	if owner.Code != http.StatusOK {
		t.Fatalf("precondition: owner read status = %d, want 200; body %s", owner.Code, owner.Body)
	}

	other := get(t, h, "m-other", p.ID)
	unknown := get(t, h, "m-other", "no-such-payment")

	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown id status = %d, want 404", unknown.Code)
	}
	if other.Code != http.StatusNotFound {
		t.Errorf("other-merchant status = %d, want 404", other.Code)
	}
	if other.Body.String() != unknown.Body.String() {
		t.Errorf("other-merchant body %q differs from unknown-id body %q; the answer must not reveal the payment exists", other.Body.String(), unknown.Body.String())
	}
	if strings.Contains(other.Body.String(), p.ID) {
		t.Errorf("other-merchant body %q leaks the payment id", other.Body.String())
	}
	var e errorJSON
	if err := json.Unmarshal(other.Body.Bytes(), &e); err != nil || e.Error == "" {
		t.Errorf("other-merchant body %q is not a JSON error with a message", other.Body.String())
	}
}

// heimdall:AC3
func TestGetPaymentByID_UnknownIDReturnsJSONError(t *testing.T) {
	h := newHandler(t)
	rec := get(t, h, "m-1", "no-such-payment")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var e errorJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil || e.Error == "" {
		t.Errorf("body %q is not a JSON error with a message", rec.Body.String())
	}
}
