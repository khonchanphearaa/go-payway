package payway

import (
	"context"
	"encoding/json"
	"fmt"
)

const pathTransactionDetails = "/api/payment-gateway/v1/payments/transaction-detail"

// TransactionDetailsRequest
type TransactionDetailsRequest struct {
	TransactionID string
}

// TransactionDetailsData is the float view of transaction-detail's data.
//
// Deprecated: floats round money. Use TransactionDetailsResponse.Transaction.
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

	// Transaction is the same data with exact amounts, identity and history.
	Transaction Transaction `json:"-"`

	// Backward-compatible convenience fields.
	TranID   string  `json:"-"`
	Status   string  `json:"-"`
	Amount   float64 `json:"-"`
	Currency string  `json:"-"`
}

func (r *TransactionDetailsResponse) apiStatus() APIStatus { return r.APIStatus }

// UnmarshalJSON fills both the legacy Data and the exact Transaction views.
func (r *TransactionDetailsResponse) UnmarshalJSON(b []byte) error {
	type plain TransactionDetailsResponse
	if err := json.Unmarshal(b, (*plain)(r)); err != nil {
		return err
	}
	var exact struct {
		Data Transaction `json:"data"`
	}
	if err := json.Unmarshal(b, &exact); err != nil {
		return err
	}
	r.Transaction = exact.Data
	return nil
}

// GetDetails returns a transaction's full details, including the merchant
// transaction_id and original_currency. PayWay limits it to 10 requests per
// minute, so do not use it for polling.
func (s *CheckoutService) GetDetails(ctx context.Context, req *TransactionDetailsRequest) (*TransactionDetailsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("payway/checkout: request is required")
	}
	var resp TransactionDetailsResponse
	if err := s.postTransaction(ctx, pathTransactionDetails, req.TransactionID, &resp); err != nil {
		return nil, err
	}

	// Populate compatibility fields for older integrations.
	resp.TranID = resp.Data.TransactionID
	resp.Status = resp.Data.PaymentStatus
	resp.Amount = resp.Data.TotalAmount
	resp.Currency = resp.Data.PaymentCurrency

	return &resp, nil
}
