package payway

import(
	"context"
	"fmt"
)

const pathCloseTransaction   = "/api/payment-gateway/v1/payments/close-transaction"

// Close transaction
type CloseTransactionResponse struct {
	Status APIStatus `json:"status"`
}

func (s *CheckoutService) CloseTransaction(ctx context.Context, tranID string) (*CloseTransactionResponse, error) {
	if tranID == "" {
		return nil, fmt.Errorf("payway/checkout: tranID is required")
	}
	if len(tranID) > 20 {
		return nil, fmt.Errorf("payway/checkout: tranID must be at most 20 characters")
	}

	reqTime := NowReqTime()
	params := map[string]string{
		"req_time":    reqTime,
		"merchant_id": s.cfg.MerchantID,
		"tran_id":     tranID,
	}

	generatedHash, err := s.hasher.GenerateOrdered(reqTime, s.cfg.MerchantID, tranID)
	if err != nil {
		return nil, err
	}
	params["hash"] = generatedHash

	var resp CloseTransactionResponse
	if err := s.http.postJSON(ctx, pathCloseTransaction, params, &resp); err != nil {
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
