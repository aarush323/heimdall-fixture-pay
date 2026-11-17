// Ticket KAN-25. Criteria: AC1, AC2, AC3.
// GET /payments lists the calling merchant's payments, reached only through
// the app.New(app.Config) seam, as the acceptance profile requires.
// Assumptions:
//   - "newest first" is judged by created_at from the injected clock, so each
//     created payment gets a strictly later time; ids are given in a different
//     order so sorting by id cannot pass.
//   - the criteria name only id, amount, currency and merchant_id, so extra
//     response fields (such as created_at) are not asserted.
//   - the ticket says nothing about a missing X-Merchant-ID header on GET, so
//     that case is not tested.
// Possible injection: none found in the ticket text.
package kan25_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aarush323/heimdall-fixture-pay/internal/app"
)

var baseTime = time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)

// advancingClock hands out a later time on every call, so creation order is
// observable in created_at.
type advancingClock struct{ next time.Time }

func (c *advancingClock) now() time.Time {
	t := c.next
	c.next = c.next.Add(time.Hour)
	return t
}

// newHandler returns a fresh service behind app.New with a deterministic clock
// and ids id_1, id_2, ...
func newHandler() http.Handler {
	clock := &advancingClock{next: baseTime}
	n := 0
	return app.New(app.Config{
		Now:   clock.now,
		NewID: func() string { n++; return "id_" + strconv.Itoa(n) },
	})
}

// newHandlerWithIDs returns a fresh service whose id generator yields exactly
// ids in order, keeping creation order independent of id order.
func newHandlerWithIDs(ids ...string) http.Handler {
	clock := &advancingClock{next: baseTime}
	n := 0
	return app.New(app.Config{
		Now: clock.now,
		NewID: func() string {
			id := ids[n]
			n++
			return id
		},
	})
}

// payment is the subset of a listed payment the criteria require.
type payment struct {
	ID         string `json:"id"`
	MerchantID string `json:"merchant_id"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
}

// create makes a payment through POST /payments and returns its id.
func create(t *testing.T, h http.Handler, merchant string, amount int64, currency string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{"amount": amount, "currency": currency})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/payments", strings.NewReader(string(body)))
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
	return p.ID
}

// list calls GET /payments as merchant, demands 200 and returns the decoded
// array together with the trimmed raw body.
func list(t *testing.T, h http.Handler, merchant string) ([]payment, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/payments", nil)
	req.Header.Set("X-Merchant-ID", merchant)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /payments status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	raw := strings.TrimSpace(rec.Body.String())
	var got []payment
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("GET /payments body %q is not a JSON array of payments: %v", raw, err)
	}
	return got, raw
}

// heimdall:AC1
func TestListPayments_ReturnsTheCallingMerchantsPayments(t *testing.T) {
	h := newHandler()
	first := create(t, h, "merchant-a", 500, "USD")
	second := create(t, h, "merchant-a", 1200, "EUR")

	got, _ := list(t, h, "merchant-a")
	if len(got) != 2 {
		t.Fatalf("list returned %d payments, want 2: %+v", len(got), got)
	}
	want := map[string]payment{
		first:  {ID: first, MerchantID: "merchant-a", Amount: 500, Currency: "USD"},
		second: {ID: second, MerchantID: "merchant-a", Amount: 1200, Currency: "EUR"},
	}
	for _, p := range got {
		w, ok := want[p.ID]
		if !ok {
			t.Errorf("unexpected payment %+v", p)
			continue
		}
		if p != w {
			t.Errorf("payment = %+v, want %+v", p, w)
		}
		delete(want, p.ID)
	}
	if len(want) != 0 {
		t.Errorf("payments missing from the list: %+v", want)
	}
}

// heimdall:AC1
func TestListPayments_ReturnsNewestFirst(t *testing.T) {
	h := newHandlerWithIDs("id_2", "id_1", "id_3")
	create(t, h, "merchant-a", 100, "USD")
	create(t, h, "merchant-a", 200, "USD")
	create(t, h, "merchant-a", 300, "USD")

	got, _ := list(t, h, "merchant-a")
	if len(got) != 3 {
		t.Fatalf("list returned %d payments, want 3: %+v", len(got), got)
	}
	wantIDs := []string{"id_3", "id_1", "id_2"}
	wantAmounts := []int64{300, 200, 100}
	for i, p := range got {
		if p.ID != wantIDs[i] || p.Amount != wantAmounts[i] {
			t.Errorf("position %d = %+v, want id %s amount %d (newest first)", i, p, wantIDs[i], wantAmounts[i])
		}
	}
}

// heimdall:AC1
func TestListPayments_ReturnsOnePaymentInsideAnArray(t *testing.T) {
	h := newHandler()
	id := create(t, h, "merchant-a", 700, "GBP")

	got, raw := list(t, h, "merchant-a")
	if len(got) != 1 {
		t.Fatalf("body %q decoded to %d payments, want exactly 1", raw, len(got))
	}
	if !strings.HasPrefix(raw, "[") {
		t.Errorf("body %q is not a JSON array", raw)
	}
	want := payment{ID: id, MerchantID: "merchant-a", Amount: 700, Currency: "GBP"}
	if got[0] != want {
		t.Errorf("payment = %+v, want %+v", got[0], want)
	}
}

// heimdall:AC2
func TestListPayments_ExcludesAnotherMerchantsPayments(t *testing.T) {
	h := newHandler()
	create(t, h, "merchant-a", 400, "USD")
	create(t, h, "merchant-a", 800, "EUR")
	foreign := create(t, h, "merchant-b", 999, "JPY")

	got, raw := list(t, h, "merchant-a")
	if len(got) != 2 {
		t.Fatalf("list returned %d payments for merchant-a, want 2: %+v", len(got), got)
	}
	for _, p := range got {
		if p.MerchantID != "merchant-a" {
			t.Errorf("payment %+v belongs to %q, want only merchant-a", p, p.MerchantID)
		}
		if p.ID == foreign {
			t.Errorf("list leaked merchant-b's payment %q", foreign)
		}
	}
	if strings.Contains(raw, foreign) || strings.Contains(raw, "merchant-b") {
		t.Errorf("body %q leaks another merchant's payment", raw)
	}
}

// heimdall:AC2
func TestListPayments_MerchantWithOnlyForeignPaymentsGetsEmptyArray(t *testing.T) {
	h := newHandler()
	create(t, h, "merchant-b", 100, "USD")
	create(t, h, "merchant-b", 200, "USD")

	got, raw := list(t, h, "merchant-a")
	if len(got) != 0 {
		t.Fatalf("body %q returned %d payments for a merchant with none, want 0", raw, len(got))
	}
	if raw != "[]" {
		t.Errorf("body = %q, want an empty JSON array []", raw)
	}
}

// heimdall:AC2
func TestListPayments_SameHandlerScopesEachMerchantsView(t *testing.T) {
	h := newHandler()
	aID := create(t, h, "merchant-a", 111, "USD")
	bID := create(t, h, "merchant-b", 222, "EUR")

	aList, aRaw := list(t, h, "merchant-a")
	if len(aList) != 1 || aList[0].ID != aID {
		t.Fatalf("merchant-a body %q, want only its payment %q", aRaw, aID)
	}

	bList, bRaw := list(t, h, "merchant-b")
	if len(bList) != 1 || bList[0].ID != bID {
		t.Fatalf("merchant-b body %q, want only its payment %q", bRaw, bID)
	}
}

// heimdall:AC3
func TestListPayments_NoPayments_ReturnsEmptyArray(t *testing.T) {
	h := newHandler()

	got, raw := list(t, h, "merchant-a")
	if len(got) != 0 {
		t.Fatalf("body %q returned %d payments, want 0", raw, len(got))
	}
	if raw != "[]" {
		t.Errorf("body = %q, want an empty JSON array []", raw)
	}
}
