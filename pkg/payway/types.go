package payway

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type Environment string

const (
	Sandbox    Environment = "sandbox"
	Production Environment = "production"
)

const (
	CurrencyUSD = "USD"
	CurrencyKHR = "KHR"
)

const (
	PaymentOptionABAPay     = "abapay"
	PaymentOptionKHQR       = "khqr"
	PaymentOptionABAPayKHQR = "abapay_khqr"
	PaymentOptionCard       = "cards"
	PaymentOptionAll        = "abapay_khqr_wechat_alipay"
)

const (
	PurchaseTypePurchase = "purchase"
	PurchaseTypePreAuth  = "pre-auth"
)

const (
	QRTemplateDefault = "template1"
	QRTemplateColor   = "template3_color"
	QRTemplateDark    = "template3_dark"
)

var baseURLs = map[Environment]string{
	Sandbox:    "https://checkout-sandbox.payway.com.kh",
	Production: "https://checkout.payway.com.kh",
}

type Config struct {
	MerchantID string
	APIKey     string
	Sandbox    bool

	// HTTPTimeout overrides the default HTTP client timeout (default: 30s)
	HTTPTimeout time.Duration
}

// Validate checks that the required config fields are present.
func (c Config) Validate() error {
	if c.MerchantID == "" {
		return fmt.Errorf("payway: MerchantID is required")
	}
	if c.APIKey == "" {
		return fmt.Errorf("payway: APIKey is required")
	}
	return nil
}

// environment returns the resolved environment
func (c Config) environment() Environment {
	if c.Sandbox {
		return Sandbox
	}
	return Production
}

// baseURL returns the correct base API URL for this config
func (c Config) baseURL() string {
	return baseURLs[c.environment()]
}

// APIStatus is returned in every PayWay response to indicate success or failure
type APIStatus struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"trace_id"`
}

// UnmarshalJSON handles both string and numeric code values from the API
func (s *APIStatus) UnmarshalJSON(data []byte) error {
	type Alias APIStatus
	aux := &struct {
		Code interface{} `json:"code"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle code as either string or numeric
	switch v := aux.Code.(type) {
	case string:
		s.Code = v
	case float64:
		s.Code = strconv.FormatInt(int64(v), 10)
	case nil:
		s.Code = ""
	default:
		return fmt.Errorf("invalid type for code: %T", v)
	}
	return nil
}

// IsSuccess returns true when PayWay signals the request succeeded
func (s APIStatus) IsSuccess() bool {
	return s.Code == "0" || s.Code == "00"
}

// Error is a structured error returned by the SDK.
type Error struct {
	Code    string
	Message string
	TraceID string
}

func (e *Error) Error() string {
	return fmt.Sprintf("payway error [%s]: %s (trace: %s)", e.Code, e.Message, e.TraceID)
}

// ReqTime formats YYYYMMDDHHmmss.
func ReqTime(t time.Time) string {
	return t.Format("20060102150405")
}

// NowReqTime returns the current UTC time in PayWay format.
func NowReqTime() string {
	return ReqTime(time.Now().UTC())
}
