package payway

import (
	"context"
	"fmt"
	"strings"

	"github.com/khonchanphearaa/go-payway/pkg/encoder"
	"github.com/khonchanphearaa/go-payway/pkg/hash"
)

const pathGenerateQR = "/api/payment-gateway/v1/payments/generate-qr"

// QRService handles all QR-related API calls
type QRService struct {
	cfg    Config
	http   *httpClient
	hasher *hash.Generator
}

// QRRequest contains all parameters for generating a PayWay QR code
type QRRequest struct {
	TransactionID   string
	Amount          float64
	Currency        string
	PaymentOption   string
	Items           []encoder.Item
	FirstName       string
	LastName        string
	Email           string
	Phone           string
	CallbackURL     string
	ReturnDeeplink  string
	Lifetime        int
	QRImageTemplate string
	PurchaseType    string
	CustomFields    string
	ReturnParams    string
	Payout          string
}

// QRResponse is the parsed response from the generate-qr endpoint
type QRResponse struct {
	QRString       string    `json:"qrString"`
	QRImage        string    `json:"qrImage"` // Base64-encoded PNG data URI
	ABAPayDeeplink string    `json:"abapay_deeplink"`
	AppStore       string    `json:"app_store"`
	PlayStore      string    `json:"play_store"`
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Status         APIStatus `json:"status"`
}

// Generate calls the PayWay QR generation endpoint
func (s *QRService) Generate(ctx context.Context, req *QRRequest) (*QRResponse, error) {
	if err := s.validateQRRequest(req); err != nil {
		return nil, err
	}

	encodedItems, err := encoder.EncodeItems(req.Items)
	if err != nil {
		return nil, fmt.Errorf("payway/qr: %w", err)
	}

	reqTime := NowReqTime()

	lifetime := req.Lifetime
	if lifetime == 0 {
		lifetime = 6
	}
	purchaseType := req.PurchaseType
	if purchaseType == "" {
		purchaseType = PurchaseTypePurchase
	}
	template := req.QRImageTemplate
	if template == "" {
		template = QRTemplateColor
	}

	firstName := req.FirstName
	lastName := req.LastName
	email := req.Email
	phone := req.Phone
	callbackURL := ""
	if req.CallbackURL != "" {
		callbackURL = encoder.ToBase64(req.CallbackURL)
	}
	returnDeeplink := req.ReturnDeeplink
	customFields := req.CustomFields
	returnParams := req.ReturnParams
	payout := req.Payout

	params := map[string]string{
		"req_time":          reqTime,
		"merchant_id":       s.cfg.MerchantID,
		"tran_id":           req.TransactionID,
		"first_name":        firstName,
		"last_name":         lastName,
		"email":             email,
		"phone":             phone,
		"amount":            fmt.Sprintf("%.2f", req.Amount),
		"purchase_type":     purchaseType,
		"payment_option":    req.PaymentOption,
		"currency":          req.Currency,
		"callback_url":      callbackURL,
		"return_deeplink":   returnDeeplink,
		"custom_fields":     customFields,
		"return_params":     returnParams,
		"payout":            payout,
		"items":             encodedItems,
		"lifetime":          fmt.Sprintf("%d", lifetime),
		"qr_image_template": template,
	}

	// QR API requires exact field order for hash concatenation
	generatedHash, err := s.hasher.GenerateOrdered(
		reqTime,
		s.cfg.MerchantID,
		req.TransactionID,
		fmt.Sprintf("%.2f", req.Amount),
		encodedItems,
		firstName,
		lastName,
		email,
		phone,
		purchaseType,
		req.PaymentOption,
		callbackURL,
		returnDeeplink,
		req.Currency,
		customFields,
		returnParams,
		payout,
		fmt.Sprintf("%d", lifetime),
		template,
	)
	if err != nil {
		return nil, err
	}
	params["hash"] = generatedHash

	var resp QRResponse
	if err := s.http.postJSON(ctx, pathGenerateQR, params, &resp); err != nil {
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

func (s *QRService) validateQRRequest(req *QRRequest) error {
	if req.TransactionID == "" {
		return fmt.Errorf("payway/qr: TransactionID is required")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("payway/qr: Amount must be greater than 0")
	}
	if req.Currency == "" {
		return fmt.Errorf("payway/qr: Currency is required (use CurrencyUSD or CurrencyKHR)")
	}
	switch strings.ToUpper(req.Currency) {
	case CurrencyKHR:
		if req.Amount < 100 {
			return fmt.Errorf("payway/qr: KHR amount must be at least 100")
		}
	case CurrencyUSD:
		if req.Amount < 0.01 {
			return fmt.Errorf("payway/qr: USD amount must be at least 0.01")
		}
	}
	if req.PaymentOption == "" {
		return fmt.Errorf("payway/qr: PaymentOption is required")
	}
	if len(req.Items) == 0 {
		return fmt.Errorf("payway/qr: at least one Item is required")
	}
	return nil
}
