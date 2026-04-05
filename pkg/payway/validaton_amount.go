package payway

import (
	"fmt"
)
// Validation amount
func (s *CheckoutService) validatePurchase(req *PurchaseRequest) error {
	if req.TransactionID == "" {
		return fmt.Errorf("payway/checkout: TransactionID is required")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("payway/checkout: Amount must be greater than 0")
	}
	if req.Currency == "" {
		return fmt.Errorf("payway/checkout: Currency is required")
	}
	if req.ReturnURL == "" {
		return fmt.Errorf("payway/checkout: ReturnURL is required")
	}
	if len(req.Items) == 0 {
		return fmt.Errorf("payway/checkout: at least one Item is required")
	}
	return nil
}
