package payway

import (
	"strings"

	"github.com/khonchanpharaa/go-payway/pkg/hash"
)

type Client struct {
	QR *QRService
	Checkout *CheckoutService
	PaymentLink *PaymentLinkService
	Callback *CallbackService
}

func NewClient(cfg Config) (*Client, error) {
	cfg.MerchantID = strings.TrimSpace(cfg.MerchantID)
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	h := hash.New(cfg.APIKey)
	httpClient := newHTTPClient(cfg)

	return &Client{
		QR: &QRService{
			cfg:    cfg,
			http:   httpClient,
			hasher: h,
		},
		Checkout: &CheckoutService{
			cfg:    cfg,
			http:   httpClient,
			hasher: h,
		},
		PaymentLink: &PaymentLinkService{
			cfg:    cfg,
			http:   httpClient,
			hasher: h,
		},
		Callback: &CallbackService{
			hasher: h,
		},
	}, nil
}
