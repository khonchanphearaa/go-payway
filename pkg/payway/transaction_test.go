package payway_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/khonchanphearaa/go-payway/pkg/payway"
)

const (
	checkPath   = "/api/payment-gateway/v1/payments/check-transaction-2"
	detailsPath = "/api/payment-gateway/v1/payments/transaction-detail"
	closePath   = "/api/payment-gateway/v1/payments/close-transaction"
)

// routed answers each path with its own 200 body; unknown paths get 404.
// calls counts requests per path.
func routed(t *testing.T, bodies map[string]string, calls map[string]int) *payway.Client {
	t.Helper()
	hc := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls[r.URL.Path]++
		status, body := 200, bodies[r.URL.Path]
		if body == "" {
			status, body = 404, `{"status":{"code":"404","message":"Not Found"}}`
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	c, err := payway.NewClient(payway.Config{MerchantID: "merchant", APIKey: testKey, Sandbox: true, HTTPClient: hc})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Documented shapes: check's status.tran_id is a gateway reference and it has
// no original_currency; details carries both.
const (
	approvedCheck   = `{"status":{"code":"00","tran_id":"GATEWAY-1"},"data":{"payment_status":"APPROVED","payment_status_code":0,"original_amount":2000,"payment_amount":0.5,"payment_currency":"USD","apv":"CHECK-APV"}}`
	approvedDetails = `{"status":{"code":"00","tran_id":"GATEWAY-2"},"data":{"transaction_id":"TX1","payment_status":"APPROVED","payment_status_code":0,"original_amount":2000.00,"original_currency":"KHR","payment_amount":0.5,"payment_currency":"USD","apv":"753786","bank_ref":"100FT1","payment_type":"KHQR","transaction_operations":[{"status":"Completed","amount":2000,"transaction_date":"2026-10-21 10:00:00"}]}}`
)

func details(data string) string { return `{"status":{"code":"00"},"data":{` + data + `}}` }

func TestVerifyApprovalConfirmsWithTransactionDetails(t *testing.T) {
	calls := map[string]int{}
	c := routed(t, map[string]string{checkPath: approvedCheck, detailsPath: approvedDetails}, calls)
	v, err := c.Checkout.VerifyApproval(context.Background(), "TX1")
	if err != nil {
		t.Fatal(err)
	}
	if !v.Approved || v.TransactionID != "TX1" || !v.OriginalAmount.Equal("2000") || v.OriginalCurrency != "KHR" ||
		v.APV != "753786" || v.BankRef != "100FT1" || v.PaymentType != "KHQR" || v.PaymentCurrency != "USD" {
		t.Fatalf("%+v", v)
	}
	if calls[checkPath] != 1 || calls[detailsPath] != 1 {
		t.Fatalf("calls %v", calls)
	}
}

func TestVerifyApprovalPendingSkipsRateLimitedDetails(t *testing.T) {
	for _, data := range []string{
		`"payment_status":"PENDING","payment_status_code":2`,
		`"payment_status":"PRE-AUTH","payment_status_code":0`,
		`"payment_status":"APPROVED"`,                         // missing code is not 0
		`"payment_status":"APPROVED","payment_status_code":1`, // conflicting signals
	} {
		calls := map[string]int{}
		v, err := routed(t, map[string]string{checkPath: details(data)}, calls).Checkout.VerifyApproval(context.Background(), "TX1")
		if err != nil || v.Approved || calls[detailsPath] != 0 {
			t.Fatalf("%s: %+v %v %v", data, v, err, calls)
		}
	}
}

func TestVerifyApprovalRejectsDisagreement(t *testing.T) {
	for name, tc := range map[string]struct{ check, details string }{
		"other transaction":    {approvedCheck, details(`"transaction_id":"TX2","payment_status":"APPROVED","payment_status_code":0,"original_amount":2000,"original_currency":"KHR"`)},
		"missing identity":     {approvedCheck, details(`"payment_status":"APPROVED","payment_status_code":0,"original_amount":2000,"original_currency":"KHR"`)},
		"details not approved": {approvedCheck, details(`"transaction_id":"TX1","payment_status":"PENDING","payment_status_code":2,"original_amount":2000,"original_currency":"KHR"`)},
		"amount differs":       {approvedCheck, details(`"transaction_id":"TX1","payment_status":"APPROVED","payment_status_code":0,"original_amount":2000.01,"original_currency":"KHR"`)},
		"missing currency":     {approvedCheck, details(`"transaction_id":"TX1","payment_status":"APPROVED","payment_status_code":0,"original_amount":2000`)},
		"currencies differ":    {strings.Replace(approvedCheck, `"apv"`, `"original_currency":"USD","apv"`, 1), approvedDetails},
		"check amount invalid": {strings.Replace(approvedCheck, `"original_amount":2000`, `"original_amount":2e3`, 1), approvedDetails},
	} {
		t.Run(name, func(t *testing.T) {
			v, err := routed(t, map[string]string{checkPath: tc.check, detailsPath: tc.details}, map[string]int{}).Checkout.VerifyApproval(context.Background(), "TX1")
			if !errors.Is(err, payway.ErrInconsistentTransaction) || v != nil {
				t.Fatalf("%+v %v", v, err)
			}
		})
	}
}

func TestVerifyApprovalPropagatesTypedErrors(t *testing.T) {
	ctx := context.Background()
	_, err := routed(t, map[string]string{checkPath: `{"status":{"code":"6","message":"Transaction not found"}}`}, map[string]int{}).Checkout.VerifyApproval(ctx, "TX1")
	if !errors.Is(err, payway.ErrTransactionNotFound) {
		t.Fatal(err)
	}
	_, err = routed(t, map[string]string{checkPath: approvedCheck, detailsPath: `{"status":{"code":"429","message":"Too many requests"}}`}, map[string]int{}).Checkout.VerifyApproval(ctx, "TX1")
	if !errors.Is(err, payway.ErrRateLimited) {
		t.Fatal(err)
	}
	if _, err = routed(t, nil, map[string]int{}).Checkout.VerifyApproval(ctx, strings.Repeat("x", 21)); err == nil {
		t.Fatal("accepted oversized tran_id")
	}
}

func TestResponsesKeepLegacyViewAndAddExactView(t *testing.T) {
	c := routed(t, map[string]string{
		checkPath:   strings.Replace(approvedCheck, `"original_amount":2000`, `"original_amount":2000.0000000001`, 1),
		detailsPath: approvedDetails,
	}, map[string]int{})
	check, err := c.Checkout.CheckTransaction(context.Background(), "TX1")
	if err != nil {
		t.Fatal(err)
	}
	if !check.IsApproved() || check.Data.OriginalAmount != 2000.0000000001 || check.Transaction.OriginalAmount != "2000.0000000001" {
		t.Fatalf("%+v", check)
	}
	if check.Transaction.OriginalAmount.Equal("2000") {
		t.Fatal("exact view rounded the amount")
	}
	if check.Status.TranID != "GATEWAY-1" {
		t.Fatal(check.Status)
	}
	d, err := c.Checkout.GetDetails(context.Background(), &payway.TransactionDetailsRequest{TransactionID: "TX1"})
	if err != nil {
		t.Fatal(err)
	}
	if d.TranID != "TX1" || d.Transaction.OriginalCurrency != "KHR" || len(d.Transaction.Operations) != 1 || !d.Transaction.Operations[0].Amount.Equal("2000") {
		t.Fatalf("%+v", d)
	}
	if _, err = c.Checkout.GetDetails(context.Background(), nil); err == nil {
		t.Fatal("accepted nil request")
	}
}

func TestErrorsMatchPerEndpointCodes(t *testing.T) {
	ctx := context.Background()
	reject := func(code string) string { return `{"status":{"code":"` + code + `","message":"m"}}` }
	for _, tc := range []struct {
		name string
		call func(*payway.Client) error
		path string
		code string
		want error
	}{
		{"check 6", func(c *payway.Client) error { _, err := c.Checkout.CheckTransaction(ctx, "TX1"); return err }, checkPath, "6", payway.ErrTransactionNotFound},
		{"check 06", func(c *payway.Client) error { _, err := c.Checkout.CheckTransaction(ctx, "TX1"); return err }, checkPath, "06", payway.ErrTransactionNotFound},
		{"check 5", func(c *payway.Client) error { _, err := c.Checkout.CheckTransaction(ctx, "TX1"); return err }, checkPath, "5", payway.ErrInvalidHash},
		{"close 5", func(c *payway.Client) error { _, err := c.Checkout.CloseTransaction(ctx, "TX1"); return err }, closePath, "5", payway.ErrTransactionNotFound},
		{"close 1", func(c *payway.Client) error { _, err := c.Checkout.CloseTransaction(ctx, "TX1"); return err }, closePath, "1", payway.ErrInvalidHash},
		{"qr 6", func(c *payway.Client) error { _, err := c.QR.Generate(ctx, khqr()); return err }, "/api/payment-gateway/v1/payments/generate-qr", "6", payway.ErrNotWhitelisted},
		{"qr 403", func(c *payway.Client) error { _, err := c.QR.Generate(ctx, khqr()); return err }, "/api/payment-gateway/v1/payments/generate-qr", "403", payway.ErrDuplicateTransaction},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call(routed(t, map[string]string{tc.path: reject(tc.code)}, map[string]int{}))
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			var perr *payway.Error
			if !errors.As(err, &perr) || perr.Code != tc.code {
				t.Fatalf("%#v", err)
			}
		})
	}
	// The same code means different things per endpoint.
	_, err := routed(t, map[string]string{closePath: reject("6")}, map[string]int{}).Checkout.CloseTransaction(ctx, "TX1")
	if err == nil || errors.Is(err, payway.ErrTransactionNotFound) {
		t.Fatal(err)
	}
	_, err = routed(t, map[string]string{"/api/payment-gateway/v1/payments/generate-qr": reject("6")}, map[string]int{}).QR.Generate(ctx, khqr())
	if errors.Is(err, payway.ErrTransactionNotFound) {
		t.Fatal("QR code 6 is a whitelist error, not a missing transaction")
	}
}

func TestHTTP429IsRateLimited(t *testing.T) {
	var calls int32
	_, err := client(t, 429, `{"status":{"code":"429","message":"slow down"}}`, &calls, nil).Checkout.CheckTransaction(context.Background(), "TX1")
	var perr *payway.Error
	if !errors.Is(err, payway.ErrRateLimited) || !errors.As(err, &perr) || perr.HTTPStatus != 429 {
		t.Fatal(err)
	}
}

func TestRedirectsAreNotFollowed(t *testing.T) {
	calls := 0
	hc := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusTemporaryRedirect, Header: http.Header{"Location": {"https://attacker.example/collect"}}, Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}, nil
	})}
	c, err := payway.NewClient(payway.Config{MerchantID: "merchant", APIKey: testKey, HTTPClient: hc})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Checkout.CheckTransaction(context.Background(), "TX1")
	var perr *payway.Error
	if !errors.As(err, &perr) || perr.HTTPStatus != http.StatusTemporaryRedirect || calls != 1 {
		t.Fatalf("redirect followed or accepted: calls=%d err=%v", calls, err)
	}
	if hc.CheckRedirect != nil {
		t.Fatal("caller's http.Client was modified")
	}
}
