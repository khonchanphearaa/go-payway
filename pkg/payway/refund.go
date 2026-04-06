package payway

import (
	"context"
	"fmt"
)

const pathRefund = "/api/merchant-portal/merchant-access/online-transaction/refund"

type RefundRequest struct {
	TransactionID string
	Amount        float64
	// MerchantAuth is RSA-encrypted payload as required by ABA docs.
	// When provided, the SDK sends request_time + merchant_id + merchant_auth.
	MerchantAuth  string
}

// RefundResponse is returned by the refund endpoint
type RefundResponse struct {
	Status APIStatus `json:"status"`
}

// Refund issues a refund for a completed transaction.
func (s *CheckoutService) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("payway/checkout: request is required")
	}
	if req.MerchantAuth == "" && req.TransactionID == "" {
		return nil, fmt.Errorf("payway/checkout: TransactionID is required")
	}
	if req.TransactionID != "" && len(req.TransactionID) > 20 {
		return nil, fmt.Errorf("payway/checkout: TransactionID must be at most 20 characters")
	}

	requestTime := NowReqTime()
	params := map[string]string{}

	if req.MerchantAuth != "" {
		params["request_time"] = requestTime
		params["merchant_id"] = s.cfg.MerchantID
		params["merchant_auth"] = req.MerchantAuth

		generatedHash, err := s.hasher.GenerateOrdered(requestTime, s.cfg.MerchantID, req.MerchantAuth)
		if err != nil {
			return nil, err
		}
		params["hash"] = generatedHash
	} else {
		params["req_time"] = requestTime
		params["merchant_id"] = s.cfg.MerchantID
		params["tran_id"] = req.TransactionID
		if req.Amount > 0 {
			params["amount"] = fmt.Sprintf("%.2f", req.Amount)
		}

		parts := []string{requestTime, s.cfg.MerchantID, req.TransactionID}
		if req.Amount > 0 {
			parts = append(parts, fmt.Sprintf("%.2f", req.Amount))
		}
		generatedHash, err := s.hasher.GenerateOrdered(parts...)
		if err != nil {
			return nil, err
		}
		params["hash"] = generatedHash
	}

	var resp RefundResponse
	if err := s.http.postJSON(ctx, pathRefund, params, &resp); err != nil {
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
