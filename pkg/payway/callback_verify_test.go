package payway_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/khonchanphearaa/go-payway/pkg/payway"
)

func callbacks(t *testing.T) *payway.CallbackService {
	t.Helper()
	c, err := payway.NewClient(payway.Config{MerchantID: "merchant", APIKey: testKey})
	if err != nil {
		t.Fatal(err)
	}
	return c.Callback
}

// Each signature below is computed from a hand-written canonical string: the
// field values ordered by key, converted as PHP does.
func TestVerifyBodyCanonicalForm(t *testing.T) {
	for _, tc := range []struct {
		name, body, signed string
	}{
		{"numeric status", `{"tran_id":"TX1","status":0,"original_currency":"KHR","original_amount":2000,"bank_ref":"BANK1"}`, "BANK12000KHR0TX1"},
		{"string status, whole float", `{"tran_id":"TX1","status":"0","original_amount":2000.0,"original_currency":"KHR"}`, "2000KHR0TX1"},
		{"large whole float has no exponent", `{"tran_id":"TX1","status":0,"original_amount":2000000.0,"original_currency":"KHR"}`, "2000000KHR0TX1"},
		{"decimals, booleans and null", `{"tran_id":"TX1","status":"0","original_amount":0.01,"paid":true,"refunded":false,"return_params":null}`, "0.0110TX1"},
		{"escaped string and nested JSON", `{"tran_id":"TX1","status":"0","return_params":"{\"order_id\":\"1\"}","extra":{"a": [1, 2]}}`, `{"a":[1,2]}{"order_id":"1"}0TX1`},
		{"documented sample", `{"tran_id":"9065703303","apv":"544415","status":"0","original_amount":0.01,"original_currency":"USD","discount_amount":0,"bank_ref":"100FT40074059022","payment_type":"ABA Pay","first_name":""}`,
			"544415100FT4007405902200.01USDABA Pay09065703303"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n, err := callbacks(t).VerifyBody([]byte(tc.body), sign(tc.signed))
			if err != nil {
				t.Fatal(err)
			}
			if !n.Verified || !n.Succeeded() || n.TranID == "" {
				t.Fatalf("%+v", n)
			}
		})
	}
}

func TestVerifyBodyExposesExactFields(t *testing.T) {
	body := `{"tran_id":"TX1","status":0,"original_currency":"KHR","original_amount":2000,"payment_amount":0.5,"payment_currency":"USD","bank_ref":"BANK1","apv":"753786","new_field":"kept"}`
	n, err := callbacks(t).VerifyBody([]byte(body), sign("753786BANK1kept2000KHR0.5USD0TX1"))
	if err != nil {
		t.Fatal(err)
	}
	if n.Status != "0" || n.OriginalCurrency != "KHR" || !n.OriginalAmount.Equal("2000.00") || n.PaymentAmount != "0.5" ||
		n.BankRef != "BANK1" || n.APV != "753786" || string(n.Fields["new_field"]) != `"kept"` {
		t.Fatalf("%+v", n)
	}
}

func TestVerifyBodyRejectsForgeryAndMalformedInput(t *testing.T) {
	const body = `{"tran_id":"TX1","status":0,"original_amount":2000}`
	good := sign("20000TX1")
	cb := callbacks(t)
	if _, err := cb.VerifyBody([]byte(body), good); err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		body, sig string
		want      error
	}{
		"tampered amount":  {strings.Replace(body, "2000", "20000", 1), good, payway.ErrInvalidSignature},
		"missing":          {body, "", payway.ErrInvalidSignature},
		"not base64":       {body, "!!" + good[2:], payway.ErrInvalidSignature},
		"oversized sig":    {body, good + good, payway.ErrInvalidSignature},
		"hex not base64":   {body, strings.Repeat("ab", 64), payway.ErrInvalidSignature},
		"duplicate key":    {`{"tran_id":"TX1","tran_id":"TX2","status":0}`, sign("TX1TX20"), payway.ErrInvalidCallback},
		"trailing data":    {body + ` {}`, good, payway.ErrInvalidCallback},
		"array":            {`[1]`, good, payway.ErrInvalidCallback},
		"empty":            {``, good, payway.ErrInvalidCallback},
		"oversized body":   {`{"tran_id":"TX1","x":"` + strings.Repeat("a", payway.MaxCallbackBytes) + `"}`, good, payway.ErrInvalidCallback},
		"missing tran_id":  {`{"status":0}`, sign("0"), payway.ErrInvalidCallback},
		"long tran_id":     {`{"tran_id":"` + strings.Repeat("x", 21) + `"}`, sign(strings.Repeat("x", 21)), payway.ErrInvalidCallback},
		"non-numeric amt":  {`{"tran_id":"TX1","original_amount":true}`, sign("1TX1"), payway.ErrInvalidCallback},
		"invalid number":   {`{"tran_id":"TX1","n":1e999}`, good, payway.ErrInvalidCallback},
		"unterminated obj": {`{"tran_id":"TX1"`, good, payway.ErrInvalidCallback},
	} {
		t.Run(name, func(t *testing.T) {
			if n, err := cb.VerifyBody([]byte(tc.body), tc.sig); !errors.Is(err, tc.want) || n != nil {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
}

func TestVerifyRequestReadsHeaderAndBoundsBody(t *testing.T) {
	const body = `{"tran_id":"TX1","status":0}`
	r := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body))
	r.Header.Set(payway.SignatureHeader, sign("0TX1"))
	if n, err := callbacks(t).VerifyRequest(r); err != nil || n.TranID != "TX1" {
		t.Fatal(n, err)
	}
	huge := bytes.Repeat([]byte(" "), 1<<20)
	r = httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(append([]byte(body), huge...)))
	r.Header.Set(payway.SignatureHeader, sign("0TX1"))
	if _, err := callbacks(t).VerifyRequest(r); !errors.Is(err, payway.ErrInvalidCallback) {
		t.Fatal(err)
	}
}

func TestParseUnverifiedIsMarked(t *testing.T) {
	n, err := callbacks(t).ParseUnverified([]byte(`{"tran_id":"TX1","status":"0","original_amount":100}`))
	if err != nil || n.Verified || n.TranID != "TX1" || !n.OriginalAmount.Equal("100") {
		t.Fatal(n, err)
	}
	if _, err = callbacks(t).ParseUnverified([]byte(`{"status":"0"}`)); !errors.Is(err, payway.ErrInvalidCallback) {
		t.Fatal(err)
	}
}

func FuzzVerifyBody(f *testing.F) {
	f.Add([]byte(`{"tran_id":"TX1","status":0,"original_amount":2000}`), "sig")
	f.Add([]byte(`{"a":{"b":[1,2,{"c":null}]},"tran_id":"T"}`), "")
	cb, _ := payway.NewClient(payway.Config{MerchantID: "m", APIKey: testKey})
	f.Fuzz(func(t *testing.T, body []byte, sig string) {
		n, err := cb.Callback.VerifyBody(body, sig)
		if (err == nil) != (n != nil) {
			t.Fatalf("inconsistent result %v %v", n, err)
		}
		if err == nil && !n.Verified {
			t.Fatal("verified body not marked")
		}
	})
}
