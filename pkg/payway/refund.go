package payway

import (
	"context"
	"fmt"
)

const pathRefund = "/api/merchant-portal/merchant-access/online-transaction/refund"

type RefundRequest struct {
	TransactionID string
	Amount        float64
}

// RefundResponse is returned by the refund endpoint
type RefundResponse struct {
	Status APIStatus `json:"status"`
}

// Refund issues a refund for a completed transaction.
func (s *CheckoutService) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	if req.TransactionID == "" {
		return nil, fmt.Errorf("payway/checkout: TransactionID is required")
	}

	reqTime := NowReqTime()
	params := map[string]string{
		"req_time":    reqTime,
		"merchant_id": s.cfg.MerchantID,
		"tran_id":     req.TransactionID,
	}
	if req.Amount > 0 {
		params["amount"] = fmt.Sprintf("%.2f", req.Amount)
	}

	generatedHash, err := s.hasher.Generate(params)
	if err != nil {
		return nil, err
	}
	params["hash"] = generatedHash

	var resp RefundResponse
	if err := s.http.postJSON(ctx, pathRefund, params, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
