package payway

import (
	"context"
	"fmt"
)

const pathCheckTransaction   = "/api/payment-gateway/v1/payments/check-transaction-2"

type CheckTransactionResponse struct {
	TranID    string    `json:"tran_id"`
	Status    string    `json:"status"`
	APIStatus APIStatus `json:"status_code"`
}

func (s *CheckoutService) CheckTransaction(ctx context.Context, tranID string) (*CheckTransactionResponse, error) {
	if tranID == "" {
		return nil, fmt.Errorf("payway/checkout: tranID is required")
	}

	reqTime := NowReqTime()
	params := map[string]string{
		"req_time":    reqTime,
		"merchant_id": s.cfg.MerchantID,
		"tran_id":     tranID,
	}

	generatedHash, err := s.hasher.Generate(params)
	if err != nil {
		return nil, err
	}
	params["hash"] = generatedHash

	var resp CheckTransactionResponse
	if err := s.http.postJSON(ctx, pathCheckTransaction, params, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}