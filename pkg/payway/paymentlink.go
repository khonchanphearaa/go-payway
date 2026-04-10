package payway

import (
	"context"
	"fmt"

	"github.com/khonchanphearaa/go-payway/pkg/encoder"
	"github.com/khonchanphearaa/go-payway/pkg/hash"
)

const (
	pathCreatePaymentLink     = "/api/merchant-portal/merchant-access/payment-link/create"
	pathGetPaymentLinkDetails = "/api/merchant-portal/merchant-access/payment-link/detail"
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
	// MerchantAuth is RSA-encrypted payload for doc-compliant payment-link create.
	// When provided, SDK sends request_time + merchant_id + merchant_auth.
	MerchantAuth  string
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

	reqTime := NowReqTime()
	params := map[string]string{
		"request_time":  reqTime,
		"merchant_id":   s.cfg.MerchantID,
		"merchant_auth": req.MerchantAuth,
	}

	generatedHash, err := s.hasher.GenerateOrdered(reqTime, s.cfg.MerchantID, req.MerchantAuth)
	if err != nil {
		return nil, err
	}
	params["hash"] = generatedHash

	var resp CreatePaymentLinkResponse
	if err := s.http.postForm(ctx, pathCreatePaymentLink, params, &resp); err != nil {
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
	if len(tranID) > 20 {
		return nil, fmt.Errorf("payway/paymentlink: tranID must be at most 20 characters")
	}

	reqTime := NowReqTime()
	params := map[string]string{
		"request_time": reqTime,
		"merchant_id":  s.cfg.MerchantID,
		"tran_id":      tranID,
	}

	generatedHash, err := s.hasher.GenerateOrdered(reqTime, s.cfg.MerchantID, tranID)
	if err != nil {
		return nil, err
	}
	params["hash"] = generatedHash

	var resp map[string]any
	if err := s.http.postJSON(ctx, pathGetPaymentLinkDetails, params, &resp); err != nil {
		return nil, err
	}

	if statusRaw, ok := resp["status"].(map[string]any); ok {
		status := APIStatus{}
		if v, ok := statusRaw["code"]; ok {
			switch c := v.(type) {
			case string:
				status.Code = c
			case float64:
				status.Code = fmt.Sprintf("%.0f", c)
			}
		}
		if v, ok := statusRaw["message"].(string); ok {
			status.Message = v
		}
		if !status.IsSuccess() {
			return nil, &Error{Code: status.Code, Message: status.Message}
		}
	}

	return resp, nil
}

// GetDetailsByMerchantAuth retrieves payment-link details using doc-compliant merchant_auth.
func (s *PaymentLinkService) GetDetailsByMerchantAuth(ctx context.Context, merchantAuth string) (map[string]any, error) {
	if merchantAuth == "" {
		return nil, fmt.Errorf("payway/paymentlink: merchantAuth is required")
	}

	requestTime := NowReqTime()
	params := map[string]string{
		"request_time":  requestTime,
		"merchant_id":   s.cfg.MerchantID,
		"merchant_auth": merchantAuth,
	}

	generatedHash, err := s.hasher.GenerateOrdered(requestTime, s.cfg.MerchantID, merchantAuth)
	if err != nil {
		return nil, err
	}
	params["hash"] = generatedHash

	var resp map[string]any
	if err := s.http.postJSON(ctx, pathGetPaymentLinkDetails, params, &resp); err != nil {
		return nil, err
	}

	if statusRaw, ok := resp["status"].(map[string]any); ok {
		status := APIStatus{}
		if v, ok := statusRaw["code"]; ok {
			switch c := v.(type) {
			case string:
				status.Code = c
			case float64:
				status.Code = fmt.Sprintf("%.0f", c)
			}
		}
		if v, ok := statusRaw["message"].(string); ok {
			status.Message = v
		}
		if !status.IsSuccess() {
			return nil, &Error{Code: status.Code, Message: status.Message}
		}
	}

	return resp, nil
}

func (s *PaymentLinkService) validateCreate(req *CreatePaymentLinkRequest) error {
	if req == nil {
		return fmt.Errorf("payway/paymentlink: request is required")
	}
	if req.MerchantAuth == "" {
		return fmt.Errorf("payway/paymentlink: MerchantAuth is required by Payment Link API")
	}
	return nil
}
