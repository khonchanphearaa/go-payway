package payway

import (
	"context"
	"fmt"

	"github.com/khonchanpharaa/go-payway/pkg/encoder"
	"github.com/khonchanpharaa/go-payway/pkg/hash"
)

const (
	pathCreatePaymentLink     = "/api/payment-gateway/v1/payments/payment-link"
	pathGetPaymentLinkDetails = "/api/payment-gateway/v1/payments/payment-link-details"
)

// PaymentLinkService handles payment link creation and retrieval.
type PaymentLinkService struct {
	cfg    Config
	http   *httpClient
	hasher *hash.Generator
}

// CreatePaymentLinkRequest describes a new payment link.
type CreatePaymentLinkRequest struct {
	TransactionID string
	Amount        float64
	Currency      string
	PaymentOption string
	Items         []encoder.Item

	// Customer info (optional).
	FirstName string
	LastName  string
	Email     string
	Phone     string

	// CallbackURL receives server-to-server notifications (plain URL).
	CallbackURL string

	// Lifetime is how many minutes the link is valid.
	Lifetime int
}

// CreatePaymentLinkResponse contains the generated payment link.
type CreatePaymentLinkResponse struct {
	PaymentLink string    `json:"payment_link"`
	Status      APIStatus `json:"status"`
}

// Create generates a shareable payment link that customers can open to pay.
func (s *PaymentLinkService) Create(ctx context.Context, req *CreatePaymentLinkRequest) (*CreatePaymentLinkResponse, error) {
	if err := s.validateCreate(req); err != nil {
		return nil, err
	}

	encodedItems, err := encoder.EncodeItems(req.Items)
	if err != nil {
		return nil, fmt.Errorf("payway/paymentlink: %w", err)
	}

	reqTime := NowReqTime()
	lifetime := req.Lifetime
	if lifetime == 0 {
		lifetime = 60
	}

	params := map[string]string{
		"req_time":       reqTime,
		"merchant_id":    s.cfg.MerchantID,
		"tran_id":        req.TransactionID,
		"amount":         fmt.Sprintf("%.2f", req.Amount),
		"currency":       req.Currency,
		"items":          encodedItems,
		"payment_option": req.PaymentOption,
		"first_name":     req.FirstName,
		"last_name":      req.LastName,
		"email":          req.Email,
		"phone":          req.Phone,
		"callback_url":   encoder.ToBase64(req.CallbackURL),
		"lifetime":       fmt.Sprintf("%d", lifetime),
	}

	generatedHash, err := s.hasher.Generate(params)
	if err != nil {
		return nil, err
	}
	params["hash"] = generatedHash

	var resp CreatePaymentLinkResponse
	if err := s.http.postJSON(ctx, pathCreatePaymentLink, params, &resp); err != nil {
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

// GetDetails retrieves the status and details of a payment link by transaction ID.
func (s *PaymentLinkService) GetDetails(ctx context.Context, tranID string) (map[string]any, error) {
	if tranID == "" {
		return nil, fmt.Errorf("payway/paymentlink: tranID is required")
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

	var resp map[string]any
	if err := s.http.postJSON(ctx, pathGetPaymentLinkDetails, params, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *PaymentLinkService) validateCreate(req *CreatePaymentLinkRequest) error {
	if req.TransactionID == "" {
		return fmt.Errorf("payway/paymentlink: TransactionID is required")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("payway/paymentlink: Amount must be greater than 0")
	}
	if req.Currency == "" {
		return fmt.Errorf("payway/paymentlink: Currency is required")
	}
	if len(req.Items) == 0 {
		return fmt.Errorf("payway/paymentlink: at least one Item is required")
	}
	return nil
}
