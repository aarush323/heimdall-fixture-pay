package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/aarush323/heimdall-fixture-pay/internal/payments"
)

// createRefundJSON is the body of POST /payments/{id}/refunds.
type createRefundJSON struct {
	Amount int64 `json:"amount"`
}

// refundJSON is the wire form of a refund.
type refundJSON struct {
	ID        string `json:"id"`
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
}

// createRefund handles POST /payments/{id}/refunds for the merchant named in the header.
func createRefund(svc *payments.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		merchant := r.Header.Get(merchantHeader)
		if merchant == "" {
			writeError(w, http.StatusUnauthorized, "missing X-Merchant-ID header")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var req createRefundJSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "body must be JSON with amount")
			return
		}
		refund, err := svc.Refund(merchant, r.PathValue("id"), req.Amount)
		switch {
		case errors.Is(err, payments.ErrNotFound):
			writeError(w, http.StatusNotFound, "payment not found")
			return
		case errors.Is(err, payments.ErrExceedsPayment):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		case err != nil:
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, refundJSON{ID: refund.ID, PaymentID: refund.PaymentID, Amount: refund.Amount})
	}
}
