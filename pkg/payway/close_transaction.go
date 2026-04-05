package payway

import(
	"context"
	"fmt"
)

// Close transaction
type CloseTransactionResponse struct {
	Status APIStatus `json:"status"`
}

func (s *CheckoutService) CloseTransaction(ctx context.Context, tranID string) (*CloseTransactionResponse, error) {
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

	var resp CloseTransactionResponse
	if err := s.http.postJSON(ctx, pathCloseTransaction, params, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
