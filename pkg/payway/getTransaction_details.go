package payway

import (
	"context"
	"fmt"
)

const pathTransactionDetails = "/api/payment-gateway/v1/payments/transaction-detail"

// TransactionDetailsRequest
type TransactionDetailsRequest struct {
	TransactionID string
}

type TransactionDetailsData struct {
	TransactionID    string  `json:"transaction_id"`
	PaymentStatus    string  `json:"payment_status"`
	TotalAmount      float64 `json:"total_amount"`
	OriginalAmount   float64 `json:"original_amount"`
	PaymentAmount    float64 `json:"payment_amount"`
	PaymentCurrency  string  `json:"payment_currency"`
	OriginalCurrency string  `json:"original_currency"`
	PaymentType      string  `json:"payment_type"`
	TransactionDate  string  `json:"transaction_date"`
}

type TransactionDetailsResponse struct {
	Data      TransactionDetailsData `json:"data"`
	APIStatus APIStatus              `json:"status"`

	// Backward-compatible convenience fields.
	TranID   string  `json:"-"`
	Status   string  `json:"-"`
	Amount   float64 `json:"-"`
	Currency string  `json:"-"`
}

func (s *CheckoutService) GetDetails(ctx context.Context, req *TransactionDetailsRequest) (*TransactionDetailsResponse, error) {
	if req.TransactionID == "" {
		return nil, fmt.Errorf("payway/checkout: TransactionID is required")
	}
	if len(req.TransactionID) > 20 {
		return nil, fmt.Errorf("payway/checkout: TransactionID must be at most 20 characters")
	}

	reqTime := NowReqTime()
	params := map[string]string{
		"req_time":    reqTime,
		"merchant_id": s.cfg.MerchantID,
		"tran_id":     req.TransactionID,
	}

	generatedHash, err := s.hasher.GenerateOrdered(reqTime, s.cfg.MerchantID, req.TransactionID)
	if err != nil {
		return nil, err
	}
	params["hash"] = generatedHash

	var resp TransactionDetailsResponse
	if err := s.http.postJSON(ctx, pathTransactionDetails, params, &resp); err != nil {
		return nil, err
	}

	if !resp.APIStatus.IsSuccess() {
		return nil, &Error{
			Code:    resp.APIStatus.Code,
			Message: resp.APIStatus.Message,
			TraceID: resp.APIStatus.TraceID,
		}
	}

	// Populate compatibility fields for older integrations.
	resp.TranID = resp.Data.TransactionID
	resp.Status = resp.Data.PaymentStatus
	resp.Amount = resp.Data.TotalAmount
	resp.Currency = resp.Data.PaymentCurrency

	return &resp, nil
}
