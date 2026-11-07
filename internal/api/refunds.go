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

// refundJSON is the wire form of a refund, as documented in api/openapi.yaml.
type refundJSON struct {
	ID        string `json:"id"`
	PaymentID string `json:"payment_id"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
}

// createRefund handles POST /payments/{id}/refunds.
func createRefund(svc *payments.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(merchantHeader) == "" {
			writeError(w, http.StatusUnauthorized, "missing X-Merchant-ID header")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var req createRefundJSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "body must be JSON with amount")
			return
		}
		refund, err := svc.Refund(r.PathValue("id"), req.Amount)
		switch {
		case errors.Is(err, payments.ErrPaymentNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, payments.ErrRefundTooLarge):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		case err != nil:
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeJSON(w, http.StatusCreated, refundJSON{
				ID:        refund.ID,
				PaymentID: refund.PaymentID,
				Amount:    refund.Amount,
				Currency:  refund.Currency,
			})
		}
	}
}

// refundCount handles GET /payments/{id}/refunds/count.
func refundCount(svc *payments.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(merchantHeader) == "" {
			writeError(w, http.StatusUnauthorized, "missing X-Merchant-ID header")
			return
		}
		n, err := svc.RefundCount(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]int{"count": n})
	}
}
