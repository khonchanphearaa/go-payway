package payway_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/khonchanphearaa/go-payway/pkg/encoder"
	"github.com/khonchanphearaa/go-payway/pkg/payway"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

const testKey = "test-api-key"

// client returns a client whose requests are answered by status/body and
// counted in calls. The last JSON request body is stored in sent.
func client(t *testing.T, status int, body string, calls *int32, sent *map[string]string) *payway.Client {
	t.Helper()
	hc := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		atomic.AddInt32(calls, 1)
		if sent != nil && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			raw, _ := io.ReadAll(r.Body)
			m := map[string]string{}
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
			*sent = m
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	c, err := payway.NewClient(payway.Config{MerchantID: "merchant", APIKey: testKey, Sandbox: true, HTTPClient: hc})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func sign(s string) string {
	m := hmac.New(sha512.New, []byte(testKey))
	m.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(m.Sum(nil))
}

func khqr() *payway.QRRequest {
	return &payway.QRRequest{TransactionID: "TX1", Amount: 100, Currency: payway.CurrencyKHR, PaymentOption: payway.PaymentOptionABAPayKHQR, Lifetime: 3}
}

func TestGenerateQRSendsDocumentedFieldsAndHash(t *testing.T) {
	var calls int32
	var sent map[string]string
	c := client(t, 200, `{"status":{"code":"0"},"qrString":"000201","amount":100,"currency":"KHR"}`, &calls, &sent)
	resp, err := c.QR.Generate(context.Background(), khqr())
	if err != nil || resp.QRString != "000201" {
		t.Fatal(resp, err)
	}
	if sent["payment_option"] != "abapay_khqr" || sent["lifetime"] != "3" || sent["qr_image_template"] != payway.QRTemplateColor || sent["amount"] != "100.00" {
		t.Fatalf("%v", sent)
	}
	// PayWay's documented order: req_time, merchant_id, tran_id, amount, items,
	// first_name, last_name, email, phone, purchase_type, payment_option,
	// callback_url, return_deeplink, currency, custom_fields, return_params,
	// payout, lifetime, qr_image_template.
	want := sign(sent["req_time"] + "merchantTX1100.00" + sent["items"] + "purchaseabapay_khqrKHR3template3_color")
	if sent["hash"] != want {
		t.Fatal("hash does not follow PayWay's field order")
	}
}

func TestGenerateQRRejectsInvalidOptionsBeforeSending(t *testing.T) {
	for name, mutate := range map[string]func(*payway.QRRequest){
		"checkout khqr option": func(r *payway.QRRequest) { r.PaymentOption = payway.PaymentOptionKHQR },
		"wechat in KHR":        func(r *payway.QRRequest) { r.PaymentOption = payway.PaymentOptionWeChat },
		"short lifetime":       func(r *payway.QRRequest) { r.Lifetime = 2 },
		"KHR below minimum":    func(r *payway.QRRequest) { r.Amount = 99 },
	} {
		t.Run(name, func(t *testing.T) {
			var calls int32
			r := khqr()
			mutate(r)
			if _, err := client(t, 200, `{}`, &calls, nil).QR.Generate(context.Background(), r); err == nil || calls != 0 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}

func TestHTTPErrorReturnsStatusWithoutBody(t *testing.T) {
	var calls int32
	body := `{"status":{"code":"04","message":"The given data was invalid.","trace_id":"T1"},"private":"raw-provider-body"}`
	_, err := client(t, 400, body, &calls, nil).QR.Generate(context.Background(), khqr())
	var perr *payway.Error
	if !errors.As(err, &perr) || perr.HTTPStatus != 400 || perr.Code != payway.CodeInvalidData || perr.TraceID != "T1" {
		t.Fatalf("%#v", err)
	}
	if strings.Contains(err.Error(), "raw-provider-body") {
		t.Fatal("raw response body leaked into error")
	}
	_, err = client(t, 502, `<not json> raw-provider-body`, &calls, nil).QR.Generate(context.Background(), khqr())
	if !errors.As(err, &perr) || perr.HTTPStatus != 502 || strings.Contains(err.Error(), "raw-provider-body") {
		t.Fatalf("%v", err)
	}
	_, err = client(t, 200, `not json raw-provider-body`, &calls, nil).QR.Generate(context.Background(), khqr())
	if err == nil || strings.Contains(err.Error(), "raw-provider-body") {
		t.Fatalf("%v", err)
	}
}

func TestCheckTransactionNotFoundIsTyped(t *testing.T) {
	var calls int32
	_, err := client(t, 200, `{"status":{"code":"6","message":"tran_id not found"}}`, &calls, nil).Checkout.CheckTransaction(context.Background(), "TX1")
	var perr *payway.Error
	if !errors.As(err, &perr) || perr.Code != payway.CodeTransactionNotFound {
		t.Fatalf("%v", err)
	}
}

func TestPurchaseSendsOnce(t *testing.T) {
	var calls int32
	c := client(t, 200, `{"status":{"code":"0"},"checkout_url":"https://pay.example/1"}`, &calls, nil)
	resp, err := c.Checkout.Purchase(context.Background(), &payway.PurchaseRequest{
		TransactionID: "TX1", Amount: 1, Currency: payway.CurrencyUSD, PaymentOption: payway.PaymentOptionABAPay,
		ReturnURL: "https://shop.example/return", Items: []encoder.Item{{Name: "Item", Quantity: 1, Price: 1}},
	})
	if err != nil || resp.CheckoutURL != "https://pay.example/1" || calls != 1 {
		t.Fatalf("resp=%v err=%v calls=%d", resp, err, calls)
	}
}

func TestOversizedResponseRejected(t *testing.T) {
	var calls int32
	body := `{"status":{"code":"0"},"qrString":"` + strings.Repeat("x", 2<<20) + `"}`
	if _, err := client(t, 200, body, &calls, nil).QR.Generate(context.Background(), khqr()); err == nil {
		t.Fatal("accepted oversized response")
	}
}

func TestHTTPClientIsCopied(t *testing.T) {
	hc := &http.Client{Timeout: 5 * time.Second}
	if _, err := payway.NewClient(payway.Config{MerchantID: "m", APIKey: "k", HTTPClient: hc, HTTPTimeout: time.Second}); err != nil {
		t.Fatal(err)
	}
	if hc.Timeout != 5*time.Second {
		t.Fatal("caller's http.Client was modified")
	}
}
