package payway

import (
	"context"
	"fmt"
)

const pathCheckTransaction = "/api/payment-gateway/v1/payments/check-transaction-2"

type CheckTransactionData struct {
	PaymentStatusCode int     `json:"payment_status_code"`
	PaymentStatus     string  `json:"payment_status"` // "APPROVED", "PENDING", etc.
	TotalAmount       float64 `json:"total_amount"`
	OriginalAmount    float64 `json:"original_amount"`
	PaymentAmount     float64 `json:"payment_amount"`
	PaymentCurrency   string  `json:"payment_currency"`
	APV               string  `json:"apv"`
	TransactionDate   string  `json:"transaction_date"`
}

type CheckTransactionResponse struct {
	Data   CheckTransactionData `json:"data"`
	Status APIStatus            `json:"status"`
}

// IsApproved returns true when payment is confirmed
func (r *CheckTransactionResponse) IsApproved() bool {
	return r.Data.PaymentStatus == "APPROVED"
}

func (s *CheckoutService) CheckTransaction(ctx context.Context, tranID string) (*CheckTransactionResponse, error) {
	if tranID == "" {
		return nil, fmt.Errorf("payway/checkout: tranID is required")
	}
	if len(tranID) > 20 {
		return nil, fmt.Errorf("payway/checkout: tranID must be at most 20 characters")
	}

	reqTime := NowReqTime()

	generatedHash, err := s.hasher.GenerateOrdered(reqTime, s.cfg.MerchantID, tranID)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"req_time":    reqTime,
		"merchant_id": s.cfg.MerchantID,
		"tran_id":     tranID,
		"hash":        generatedHash,
	}

	var resp CheckTransactionResponse
	if err := s.http.postJSON(ctx, pathCheckTransaction, params, &resp); err != nil {
		return nil, err
	}

	if !resp.Status.IsSuccess() {
		return nil, &Error{
			Code:    resp.Status.Code,
			Message: resp.Status.Message,
			TraceID: resp.Status.TraceID,
		}
	}

	return &resp, nil
}
