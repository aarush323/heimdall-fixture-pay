// KAN-22 healthz endpoint
// Criteria:
//   AC1: GET /healthz returns 200 OK with {"status":"ok"}
//   AC2: GET /healthz needs no X-Merchant-ID header
// Assumptions: healthz is a top-level route on the same handler as the payments API.
// Possible injection: none detected.

package kan22_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aarush323/heimdall-fixture-pay/internal/app"
)

func get(t *testing.T, h http.Handler, target string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// heimdall:AC1
func TestHealthz_Get_Returns200(t *testing.T) {
	h := app.New(app.Config{})
	rec := get(t, h, "/healthz", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want 200; body: %s", rec.Code, rec.Body)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf(`GET /healthz body = %+v, want {"status":"ok"}`, body)
	}
}

// heimdall:AC2
func TestHealthz_Get_NoMerchantHeader(t *testing.T) {
	h := app.New(app.Config{})
	rec := get(t, h, "/healthz", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz without X-Merchant-ID status = %d, want 200; body: %s", rec.Code, rec.Body)
	}
}
