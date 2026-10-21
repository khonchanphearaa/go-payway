package payway

import (
	"context"
)

const pathCloseTransaction = "/api/payment-gateway/v1/payments/close-transaction"

// CloseTransactionResponse is close-transaction's reply; Status.TranID is the
// closed transaction.
type CloseTransactionResponse struct {
	Status APIStatus `json:"status"`
}

func (r *CloseTransactionResponse) apiStatus() APIStatus { return r.Status }

// CloseTransaction cancels a transaction before it is paid. PayWay sends no
// callback afterwards. An unknown tranID returns an error matching
// ErrTransactionNotFound (close uses status code 5, not 6).
func (s *CheckoutService) CloseTransaction(ctx context.Context, tranID string) (*CloseTransactionResponse, error) {
	var resp CloseTransactionResponse
	if err := s.postTransaction(ctx, pathCloseTransaction, tranID, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
