package payway

import (
	"context"
	"fmt"
)

// TransactionDetailsRequest
type TransactionDetailsRequest struct {
	TransactionID string
}

type TransactionDetailsResponse struct {
	TranID        string    `json:"tran_id"`
	Status        string    `json:"status"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	PaymentOption string    `json:"payment_option"`
	APIStatus     APIStatus `json:"status_code"`
}

func (s *CheckoutService) GetDetails(ctx context.Context, req *TransactionDetailsRequest) (*TransactionDetailsResponse, error) {
	if req.TransactionID == "" {
		return nil, fmt.Errorf("payway/checkout: TransactionID is required")
	}

	reqTime := NowReqTime()
	params := map[string]string{
		"req_time":    reqTime,
		"merchant_id": s.cfg.MerchantID,
		"tran_id":     req.TransactionID,
	}

	generatedHash, err := s.hasher.Generate(params)
	if err != nil {
		return nil, err
	}
	params["hash"] = generatedHash

	var resp TransactionDetailsResponse
	if err := s.http.postJSON(ctx, pathTransactionDetails, params, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
