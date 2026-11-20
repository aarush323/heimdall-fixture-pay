// KAN-26 per-payment cap of 1,000,000 in the smallest currency unit
// Criteria:
//   AC1: POST /payments with amount 1000000 (the cap) returns 201 and the created payment.
//   AC2: POST /payments with amount 1000001 returns 400 with a JSON body {"error": ...}
//        whose message names the amount and the cap 1000000.
//   AC3: The cap applies in every currency: amount 1000001 with currency JPY also returns 400.
// Assumptions: the cap is inclusive (1000000 accepted, 1000001 refused); the 400 message
// contains the amount and the cap as plain decimal digits (no grouping separators); the
// service is driven only through the app.New seam with an injected clock and id source.
// The at-cap cases already hold on base, so each is joined to an above-cap case in the
// same test so the whole test fails red on base.
// Possible injection: none detected.

package kan26_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aarush323/heimdall-fixture-pay/internal/app"
)

var fixedNow = time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC)

type paymentBody struct {
	ID         string    `json:"id"`
	MerchantID string    `json:"merchant_id"`
	Amount     int64     `json:"amount"`
	Currency   string    `json:"currency"`
	CreatedAt  time.Time `json:"created_at"`
}

type errorBody struct {
	Error string `json:"error"`
}

// newHandler builds the service through the seam with a fixed clock and counting ids.
func newHandler() http.Handler {
	ids := []string{"id_1", "id_2", "id_3", "id_4"}
	var n int
	return app.New(app.Config{
		Now: func() time.Time { return fixedNow },
		NewID: func() string {
			id := ids[n%len(ids)]
			n++
			return id
		},
	})
}

// postPayment sends JSON to POST /payments with the merchant header set.
func postPayment(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Merchant-ID", "m-1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// heimdall:AC1
// The cap boundary in USD: one over the cap is refused, the cap itself is accepted
// and the created payment is returned.
func TestCreatePayment_USD_RefusesAboveCapAndAcceptsAtCap(t *testing.T) {
	h := newHandler()

	over := postPayment(t, h, `{"amount": 1000001, "currency": "USD"}`)
	if over.Code != http.StatusBadRequest {
		t.Fatalf("POST /payments amount 1000001 status = %d, want 400; body %s", over.Code, over.Body)
	}

	rec := postPayment(t, h, `{"amount": 1000000, "currency": "USD"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /payments amount 1000000 status = %d, want 201; body %s", rec.Code, rec.Body)
	}
	var got paymentBody
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("201 body is not a payment: %v; body %s", err, rec.Body)
	}
	if got.ID == "" {
		t.Errorf("created payment has no id: %+v", got)
	}
	if got.MerchantID != "m-1" {
		t.Errorf("merchant_id = %q, want m-1", got.MerchantID)
	}
	if got.Amount != 1000000 {
		t.Errorf("amount = %d, want 1000000", got.Amount)
	}
	if got.Currency != "USD" {
		t.Errorf("currency = %q, want USD", got.Currency)
	}
	if !got.CreatedAt.Equal(fixedNow) {
		t.Errorf("created_at = %v, want the injected %v", got.CreatedAt, fixedNow)
	}
}

// heimdall:AC2
// One over the cap returns 400 with an error message naming the amount and the cap.
func TestCreatePayment_AboveCap_Returns400NamingAmountAndCap(t *testing.T) {
	rec := postPayment(t, newHandler(), `{"amount": 1000001, "currency": "USD"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST /payments amount 1000001 status = %d, want 400; body %s", rec.Code, rec.Body)
	}
	var e errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("400 body is not JSON with an error field: %v; body %s", err, rec.Body)
	}
	if e.Error == "" {
		t.Fatalf("400 body %q does not carry {\"error\": ...}", rec.Body)
	}
	if !strings.Contains(e.Error, "1000001") {
		t.Errorf("error %q does not name the amount 1000001", e.Error)
	}
	if !strings.Contains(e.Error, "1000000") {
		t.Errorf("error %q does not name the cap 1000000", e.Error)
	}
}

// heimdall:AC2
// The refusal is not special-cased to USD: one over the cap in EUR is refused too.
func TestCreatePayment_JustAboveCapInEUR_Returns400(t *testing.T) {
	rec := postPayment(t, newHandler(), `{"amount": 1000001, "currency": "EUR"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST /payments amount 1000001 currency EUR status = %d, want 400; body %s", rec.Code, rec.Body)
	}
	var e errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("400 body is not JSON with an error field: %v; body %s", err, rec.Body)
	}
	if e.Error == "" {
		t.Errorf("400 body %q does not carry {\"error\": ...}", rec.Body)
	}
}

// heimdall:AC3
// The cap applies in every currency: JPY one over the cap is refused and the cap itself
// is accepted, so the limit does not depend on the currency.
func TestCreatePayment_JPY_RefusesAboveCapAndAcceptsAtCap(t *testing.T) {
	h := newHandler()

	over := postPayment(t, h, `{"amount": 1000001, "currency": "JPY"}`)
	if over.Code != http.StatusBadRequest {
		t.Fatalf("POST /payments amount 1000001 currency JPY status = %d, want 400; body %s", over.Code, over.Body)
	}
	var e errorBody
	if err := json.Unmarshal(over.Body.Bytes(), &e); err != nil {
		t.Fatalf("400 body is not JSON with an error field: %v; body %s", err, over.Body)
	}
	if e.Error == "" {
		t.Errorf("400 body %q does not carry {\"error\": ...}", over.Body)
	}

	rec := postPayment(t, h, `{"amount": 1000000, "currency": "JPY"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /payments amount 1000000 currency JPY status = %d, want 201; body %s", rec.Code, rec.Body)
	}
	var got paymentBody
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("201 body is not a payment: %v; body %s", err, rec.Body)
	}
	if got.Amount != 1000000 || got.Currency != "JPY" {
		t.Errorf("payment = %+v, want amount 1000000 in JPY", got)
	}
}
