package payway

import (
	"context"
	"fmt"
)

// Payment statuses reported in Transaction.PaymentStatus.
const (
	StatusApproved  = "APPROVED"
	StatusPreAuth   = "PRE-AUTH"
	StatusPending   = "PENDING"
	StatusDeclined  = "DECLINED"
	StatusRefunded  = "REFUNDED"
	StatusCancelled = "CANCELLED"
)

// Transaction is a payment as reported by check-transaction or
// transaction-detail, with exact amounts. Fields an endpoint does not return
// are empty: check-transaction omits TransactionID and OriginalCurrency.
type Transaction struct {
	// TransactionID is the merchant tran_id (transaction-detail only).
	TransactionID     string `json:"transaction_id"`
	PaymentStatus     string `json:"payment_status"`
	PaymentStatusCode *int   `json:"payment_status_code"`

	OriginalAmount   Amount `json:"original_amount"`
	OriginalCurrency string `json:"original_currency"`
	TotalAmount      Amount `json:"total_amount"`
	PaymentAmount    Amount `json:"payment_amount"`
	PaymentCurrency  string `json:"payment_currency"`
	RefundAmount     Amount `json:"refund_amount"`
	DiscountAmount   Amount `json:"discount_amount"`

	APV             string `json:"apv"`
	TransactionDate string `json:"transaction_date"`

	// Transaction-detail only.
	BankRef      string                 `json:"bank_ref"`
	PaymentType  string                 `json:"payment_type"`
	PayerAccount string                 `json:"payer_account"`
	BankName     string                 `json:"bank_name"`
	CardSource   string                 `json:"card_source"`
	Operations   []TransactionOperation `json:"transaction_operations"`
}

// TransactionOperation is one entry in a transaction's history.
type TransactionOperation struct {
	Status          string `json:"status"`
	Amount          Amount `json:"amount"`
	TransactionDate string `json:"transaction_date"`
	BankRef         string `json:"bank_ref"`
}

// Approved reports a completed payment. It requires both signals: status code
// 0 alone also covers PRE-AUTH, and a missing code is not treated as 0.
func (t Transaction) Approved() bool {
	return t.PaymentStatus == StatusApproved && t.PaymentStatusCode != nil && *t.PaymentStatusCode == 0
}

// PreAuthorized reports a pre-authorization hold that has not been captured.
func (t Transaction) PreAuthorized() bool {
	return t.PaymentStatus == StatusPreAuth && t.PaymentStatusCode != nil && *t.PaymentStatusCode == 0
}

// Verification is the result of VerifyApproval.
type Verification struct {
	Approved      bool
	TransactionID string
	// PaymentStatus is PayWay's status, for example PENDING when not approved.
	PaymentStatus string

	// Set only when Approved.
	OriginalAmount   Amount
	OriginalCurrency string
	PaymentAmount    Amount
	PaymentCurrency  string
	APV              string
	BankRef          string
	PaymentType      string
}

// VerifyApproval decides whether tranID has been paid, using only PayWay's
// authenticated APIs. Use it before fulfilling an order; a callback alone is
// never proof of payment.
//
// It polls check-transaction (600 requests/second). Check's status.tran_id is
// a gateway reference and it omits original_currency, so an approval is then
// confirmed with transaction-detail (10 requests/minute, called only for
// approvals). Both must agree on status and exact original amount, and details
// must name tranID and its original currency; otherwise the error wraps
// ErrInconsistentTransaction.
//
// A pending, declined or pre-authorized transaction returns Approved false and
// a nil error. The caller must still compare OriginalAmount and
// OriginalCurrency with what it charged.
func (s *CheckoutService) VerifyApproval(ctx context.Context, tranID string) (*Verification, error) {
	check, err := s.CheckTransaction(ctx, tranID)
	if err != nil {
		return nil, err
	}
	c := check.Transaction
	if !c.Approved() {
		return &Verification{TransactionID: tranID, PaymentStatus: c.PaymentStatus}, nil
	}
	if !c.OriginalAmount.Valid() {
		return nil, inconsistent("check-transaction original_amount is missing or invalid")
	}
	details, err := s.GetDetails(ctx, &TransactionDetailsRequest{TransactionID: tranID})
	if err != nil {
		return nil, err
	}
	d := details.Transaction
	switch {
	case d.TransactionID != tranID:
		return nil, inconsistent("transaction-detail names a different transaction")
	case !d.Approved():
		return nil, inconsistent("check and details disagree on status")
	case !d.OriginalAmount.Equal(c.OriginalAmount):
		return nil, inconsistent("check and details disagree on original amount")
	case d.OriginalCurrency == "":
		return nil, inconsistent("transaction-detail original_currency is missing")
	case c.OriginalCurrency != "" && c.OriginalCurrency != d.OriginalCurrency:
		return nil, inconsistent("check and details disagree on original currency")
	}
	apv := d.APV
	if apv == "" {
		apv = c.APV
	}
	return &Verification{
		Approved:         true,
		TransactionID:    tranID,
		PaymentStatus:    d.PaymentStatus,
		OriginalAmount:   d.OriginalAmount,
		OriginalCurrency: d.OriginalCurrency,
		PaymentAmount:    d.PaymentAmount,
		PaymentCurrency:  d.PaymentCurrency,
		APV:              apv,
		BankRef:          d.BankRef,
		PaymentType:      d.PaymentType,
	}, nil
}

func inconsistent(reason string) error {
	return fmt.Errorf("%w: %s", ErrInconsistentTransaction, reason)
}

// statusResponse is implemented by every response that carries a status envelope.
type statusResponse interface {
	apiStatus() APIStatus
}

// postTransaction sends the request shape shared by check, details and close:
// req_time, merchant_id and tran_id, signed in that order.
func (s *CheckoutService) postTransaction(ctx context.Context, path, tranID string, dest statusResponse) error {
	if err := validateTranID(tranID); err != nil {
		return err
	}
	reqTime := NowReqTime()
	generatedHash, err := s.hasher.GenerateOrdered(reqTime, s.cfg.MerchantID, tranID)
	if err != nil {
		return err
	}
	params := map[string]string{
		"req_time":    reqTime,
		"merchant_id": s.cfg.MerchantID,
		"tran_id":     tranID,
		"hash":        generatedHash,
	}
	if err := s.http.postJSON(ctx, path, params, dest); err != nil {
		return err
	}
	if status := dest.apiStatus(); !status.IsSuccess() {
		return statusError(path, status)
	}
	return nil
}

func validateTranID(tranID string) error {
	if tranID == "" {
		return fmt.Errorf("payway/checkout: tranID is required")
	}
	if len(tranID) > 20 {
		return fmt.Errorf("payway/checkout: tranID must be at most 20 characters")
	}
	return nil
}
