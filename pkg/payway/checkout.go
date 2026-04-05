package payway

import (
	"context"
	"fmt"

	"github.com/khonchanpharaa/go-payway/pkg/encoder"
	"github.com/khonchanpharaa/go-payway/pkg/hash"
)

const (
	pathPurchase           = "/api/payment-gateway/v1/payments/purchase"
	pathTransactionDetails = "/api/payment-gateway/v1/payments/details"
	pathCloseTransaction   = "/api/payment-gateway/v1/payments/close-transaction"
	pathCheckTransaction   = "/api/payment-gateway/v1/payments/check-transaction"
	pathRefund             = "/api/payment-gateway/v1/payments/refund"
	pathTransactionList    = "/api/payment-gateway/v1/payments/list"
	pathExchangeRate       = "/api/payment-gateway/v1/payments/exchange-rate"
)

type CheckoutService struct {
	cfg    Config
	http   *httpClient
	hasher *hash.Generator
}

type PurchaseRequest struct {
	TransactionID string
	Amount        float64
	Currency      string
	PaymentOption string
	Items         []encoder.Item

	// Customer info.
	FirstName string
	LastName  string
	Email     string
	Phone     string

	ReturnURL string
	CancelURL string
	CallbackURL string
	Type string

	// Optional fields.
	Shipping           string
	SkipSuccessPage    bool
	ContinueSuccessURL string
	ReturnDeeplink     string
	CustomFields       string
	ReturnParams       string
	Lifetime           int
	PurchaseType       string
}

// PurchaseResponse contains the checkout redirect URL.
type PurchaseResponse struct {
	CheckoutURL string

	Status APIStatus
}

// Purchase initiates a checkout session and returns the URL to redirect the user to.
func (s *CheckoutService) Purchase(ctx context.Context, req *PurchaseRequest) (*PurchaseResponse, error) {
	if err := s.validatePurchase(req); err != nil {
		return nil, err
	}

	encodedItems, err := encoder.EncodeItems(req.Items)
	if err != nil {
		return nil, fmt.Errorf("payway/checkout: %w", err)
	}

	reqTime := NowReqTime()
	lifetime := req.Lifetime
	if lifetime == 0 {
		lifetime = 30
	}
	purchaseType := req.PurchaseType
	if purchaseType == "" {
		purchaseType = PurchaseTypePurchase
	}
	viewType := req.Type
	if viewType == "" {
		viewType = "checkout"
	}
	skipSuccess := "0"
	if req.SkipSuccessPage {
		skipSuccess = "1"
	}

	amountStr := fmt.Sprintf("%.2f", req.Amount)

	fields := map[string]string{
		"req_time":             reqTime,
		"merchant_id":          s.cfg.MerchantID,
		"tran_id":              req.TransactionID,
		"amount":               amountStr,
		"currency":             req.Currency,
		"items":                encodedItems,
		"type":                 purchaseType,
		"payment_option":       req.PaymentOption,
		"firstname":            req.FirstName,
		"lastname":             req.LastName,
		"email":                req.Email,
		"phone":                req.Phone,
		"return_url":           encoder.ToBase64(req.ReturnURL),
		"cancel_url":           encoder.ToBase64(req.CancelURL),
		"callback_url":         encoder.ToBase64(req.CallbackURL),
		"view_type":            viewType,
		"skip_success_page":    skipSuccess,
		"continue_success_url": req.ContinueSuccessURL,
		"return_deeplink":      req.ReturnDeeplink,
		"custom_fields":        req.CustomFields,
		"return_params":        req.ReturnParams,
		"shipping":             req.Shipping,
		"lifetime":             fmt.Sprintf("%d", lifetime),
	}

	generatedHash, err := s.hasher.Generate(fields)
	if err != nil {
		return nil, err
	}
	fields["hash"] = generatedHash
	var raw map[string]any
	if err := s.http.postForm(ctx, pathPurchase, fields, &raw); err != nil {
		return nil, err
	}

	return &PurchaseResponse{
		CheckoutURL: s.cfg.baseURL() + pathPurchase,
		Status:      APIStatus{Code: "0", Message: "Success"},
	}, nil
}


// ExchangeRateResponse 
type ExchangeRateResponse struct {
	Rate   float64   `json:"rate"`
	Status APIStatus `json:"status"`
}

// GetExchangeRate fetches the current USD-to-KHR exchange rate used by PayWay.
func (s *CheckoutService) GetExchangeRate(ctx context.Context) (*ExchangeRateResponse, error) {
	reqTime := NowReqTime()
	params := map[string]string{
		"req_time":    reqTime,
		"merchant_id": s.cfg.MerchantID,
	}

	generatedHash, err := s.hasher.Generate(params)
	if err != nil {
		return nil, err
	}
	params["hash"] = generatedHash

	var resp ExchangeRateResponse
	if err := s.http.postJSON(ctx, pathExchangeRate, params, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}



// // Validation amount
// func (s *CheckoutService) validatePurchase(req *PurchaseRequest) error {
// 	if req.TransactionID == "" {
// 		return fmt.Errorf("payway/checkout: TransactionID is required")
// 	}
// 	if req.Amount <= 0 {
// 		return fmt.Errorf("payway/checkout: Amount must be greater than 0")
// 	}
// 	if req.Currency == "" {
// 		return fmt.Errorf("payway/checkout: Currency is required")
// 	}
// 	if req.ReturnURL == "" {
// 		return fmt.Errorf("payway/checkout: ReturnURL is required")
// 	}
// 	if len(req.Items) == 0 {
// 		return fmt.Errorf("payway/checkout: at least one Item is required")
// 	}
// 	return nil
// }
