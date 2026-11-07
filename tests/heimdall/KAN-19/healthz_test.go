// KAN-19: healthz endpoint.
// AC1: GET /healthz returns 200 {"status":"ok"}.
// AC2: Other methods return 405.
// Assumptions: healthz lives on Routes; no injection found.

package kan19_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aarush323/heimdall-fixture-pay/internal/api"
	"github.com/aarush323/heimdall-fixture-pay/internal/payments"
)

func newHandler() http.Handler {
	svc := payments.NewService(payments.NewMemoryStore(), func() time.Time { return time.Now() }, func() string { return "" })
	return api.Routes(svc)
}

func req(method string) *httptest.ResponseRecorder {
	h := newHandler()
	r := httptest.NewRequest(method, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

// heimdall:AC1
func TestHealthzGet_Returns200WithStatusOk(t *testing.T) {
	rec := req(http.MethodGet)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want 200; body: %s", rec.Code, rec.Body)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("GET /healthz body is not JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf(`GET /healthz status = %q, want "ok"`, body["status"])
	}
}

// heimdall:AC2
func TestHealthzPost_Returns405(t *testing.T) {
	rec := req(http.MethodPost)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /healthz status = %d, want 405; body: %s", rec.Code, rec.Body)
	}
}