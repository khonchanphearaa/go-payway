package payway

import (
	"encoding/json"
	"fmt"
	"net/http"
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

// Payment options. The QR API (QR.Generate) accepts only PaymentOptionABAPayKHQR,
// PaymentOptionWeChat and PaymentOptionAlipay; the others are checkout options.
const (
	PaymentOptionABAPay = "abapay"
	// PaymentOptionKHQR is a checkout option. PayWay's QR API rejects it with
	// code 04 ("The given data was invalid"); use PaymentOptionABAPayKHQR there.
	PaymentOptionKHQR       = "khqr"
	PaymentOptionABAPayKHQR = "abapay_khqr"
	PaymentOptionWeChat     = "wechat" // QR API, USD only
	PaymentOptionAlipay     = "alipay" // QR API, USD only
	PaymentOptionCard       = "cards"
	PaymentOptionAll        = "abapay_khqr_wechat_alipay"
)

// Status codes returned by PayWay that callers commonly branch on.
const (
	// CodeInvalidData means a request field failed PayWay validation.
	CodeInvalidData = "04"
	// CodeTransactionNotFound is check-transaction's and transaction-detail's
	// "not found" code. Close-transaction uses 5; prefer
	// errors.Is(err, ErrTransactionNotFound), which knows each endpoint.
	CodeTransactionNotFound = "6"
)

const (
	PurchaseTypePurchase = "purchase"
	PurchaseTypePreAuth  = "pre-auth"
)

// QR image templates. PayWay documents template3_color as its example value;
// render qrString yourself if you don't need PayWay's image.
const (
	QRTemplateDefault = "template1"
	QRTemplateColor   = "template3_color"
	QRTemplateDark    = "template3_dark"
	// QRTemplateDigital is not listed in PayWay's public QR API document.
	QRTemplateDigital = "template4_color"
)

var baseURLs = map[Environment]string{
	Sandbox:    "https://checkout-sandbox.payway.com.kh",
	Production: "https://checkout.payway.com.kh",
}

type Config struct {
	MerchantID string
	APIKey     string
	Sandbox    bool

	// HTTPTimeout overrides the HTTP client timeout (default: 10s, or the
	// HTTPClient's own timeout when one is set).
	HTTPTimeout time.Duration

	// HTTPClient is an optional client for transport, proxy or test settings.
	// The SDK copies it; nil uses a default client. Redirects are refused
	// unless the client sets its own CheckRedirect.
	HTTPClient *http.Client
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
	// TranID is endpoint-specific. Check-transaction and transaction-detail
	// document it as a gateway request reference, not the merchant tran_id;
	// close-transaction returns the closed tran_id. Never use it as proof of
	// which transaction a response describes.
	TranID string `json:"tran_id"`
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

// ReqTime formats YYYYMMDDHHmmss.
func ReqTime(t time.Time) string {
	return t.Format("20060102150405")
}

// NowReqTime returns the current UTC time in PayWay format.
func NowReqTime() string {
	return ReqTime(time.Now().UTC())
}
