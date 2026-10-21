package payway

import (
	"context"
	"encoding/json"
)

const pathCheckTransaction = "/api/payment-gateway/v1/payments/check-transaction-2"

// CheckTransactionData is the float view of check-transaction's data.
//
// Deprecated: floats round money. Use CheckTransactionResponse.Transaction.
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

	// Transaction is the same data with exact amounts.
	Transaction Transaction `json:"-"`
}

func (r *CheckTransactionResponse) apiStatus() APIStatus { return r.Status }

// UnmarshalJSON fills both the legacy Data and the exact Transaction views.
func (r *CheckTransactionResponse) UnmarshalJSON(b []byte) error {
	type plain CheckTransactionResponse
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

// IsApproved reports a completed payment: status APPROVED with status code 0.
// Status code 0 alone also covers PRE-AUTH.
func (r *CheckTransactionResponse) IsApproved() bool {
	return r.Data.PaymentStatus == StatusApproved && r.Data.PaymentStatusCode == 0
}

// CheckTransaction polls a payment's status. It covers transactions created in
// the last 7 days; use GetDetails for older ones. Before fulfilling an order,
// prefer VerifyApproval, which also confirms identity and currency.
func (s *CheckoutService) CheckTransaction(ctx context.Context, tranID string) (*CheckTransactionResponse, error) {
	var resp CheckTransactionResponse
	if err := s.postTransaction(ctx, pathCheckTransaction, tranID, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
