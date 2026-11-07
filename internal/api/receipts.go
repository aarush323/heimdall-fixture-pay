package api

import (
	"errors"
	"net/http"

	"github.com/aarush323/heimdall-fixture-pay/internal/payments"
)

// receiptJSON is the wire form of a refund receipt, as documented in api/openapi.yaml.
type receiptJSON struct {
	RefundID string `json:"refund_id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// getReceipt handles GET /payments/{id}/refunds/{refund_id}/receipt.
func getReceipt(svc *payments.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		receipt, err := svc.Receipt(r.PathValue("id"), r.PathValue("refund_id"))
		if errors.Is(err, payments.ErrRefundNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, receiptJSON{RefundID: receipt.RefundID, Amount: receipt.Amount, Currency: receipt.Currency})
	}
}
