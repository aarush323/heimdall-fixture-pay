package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/aarush323/heimdall-fixture-pay/internal/payments"
)

// createPaymentJSON is the body of POST /payments.
type createPaymentJSON struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// paymentJSON is the wire form of a payment, as documented in api/openapi.yaml.
type paymentJSON struct {
	ID         string    `json:"id"`
	MerchantID string    `json:"merchant_id"`
	Amount     int64     `json:"amount"`
	Currency   string    `json:"currency"`
	CreatedAt  time.Time `json:"created_at"`
}

// createPayment handles POST /payments for the merchant named in the header.
func createPayment(svc *payments.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		merchant := r.Header.Get(merchantHeader)
		if merchant == "" {
			writeError(w, http.StatusUnauthorized, "missing X-Merchant-ID header")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var req createPaymentJSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "body must be JSON with amount and currency")
			return
		}
		p, err := svc.Create(merchant, req.Amount, req.Currency)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, toJSON(p))
	}
}

// getPayment handles GET /payments/{id}: the caller's own payment, or 404 for an unknown id or another merchant's.
func getPayment(svc *payments.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		merchant := r.Header.Get(merchantHeader)
		if merchant == "" {
			writeError(w, http.StatusUnauthorized, "missing X-Merchant-ID header")
			return
		}
		p, err := svc.Get(merchant, r.PathValue("id"))
		if errors.Is(err, payments.ErrNotFound) {
			writeError(w, http.StatusNotFound, "payment not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not read the payment")
			return
		}
		writeJSON(w, http.StatusOK, toJSON(p))
	}
}

// toJSON is the wire form of p.
func toJSON(p payments.Payment) paymentJSON {
	return paymentJSON{ID: p.ID, MerchantID: p.MerchantID, Amount: p.Amount, Currency: p.Currency, CreatedAt: p.CreatedAt}
}
