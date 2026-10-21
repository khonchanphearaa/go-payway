package payway

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

const (
	// SignatureHeader carries PayWay's Base64 HMAC-SHA512 of a callback body.
	SignatureHeader = "X-PayWay-HMAC-SHA512"
	// MaxCallbackBytes bounds a callback body. PayWay notifications are well under 2 KiB.
	MaxCallbackBytes = 8 << 10
)

// Notification is a payment callback ("pushback") PayWay POSTs to callback_url.
//
// A notification is a prompt to check, not proof of payment: PayWay requires
// merchants to confirm it with the check-transaction API. Call
// Checkout.VerifyApproval before fulfilling an order.
type Notification struct {
	TranID           string
	Status           string // "0" means the payer completed payment
	APV              string
	BankRef          string
	PaymentType      string
	OriginalAmount   Amount
	OriginalCurrency string
	PaymentAmount    Amount
	PaymentCurrency  string
	TotalAmount      Amount
	DiscountAmount   Amount
	TransactionDate  string
	ReturnParams     string
	PayerAccount     string
	BankName         string
	CardSource       string

	// Verified is true when the body's signature was checked.
	Verified bool
	// Fields holds every top-level field as received, including ones this
	// SDK does not name yet.
	Fields map[string]json.RawMessage
}

// Succeeded reports whether the callback says the payer completed payment.
func (n *Notification) Succeeded() bool {
	return n.Status == "0" || n.Status == "00"
}

// VerifyRequest reads at most MaxCallbackBytes of r's body and verifies it
// against the SignatureHeader.
func (s *CallbackService) VerifyRequest(r *http.Request) (*Notification, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxCallbackBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: unreadable body", ErrInvalidCallback)
	}
	return s.VerifyBody(body, r.Header.Get(SignatureHeader))
}

// VerifyBody verifies a callback body against its X-PayWay-HMAC-SHA512 value.
//
// PayWay signs the values of every top-level field, ordered by key, the way
// PHP converts them to strings: strings as-is, whole numbers without a
// decimal point (2000.0 signs as "2000"), true as "1", false and null as "".
// The signature is compared in constant time. Duplicate keys and data after
// the JSON object are rejected, so the signed and parsed fields cannot differ.
func (s *CallbackService) VerifyBody(body []byte, signature string) (*Notification, error) {
	fields, err := callbackFields(body)
	if err != nil {
		return nil, err
	}
	signed, err := signedCallbackString(fields)
	if err != nil {
		return nil, err
	}
	if !s.hasher.Equal([]byte(signed), signature) {
		return nil, ErrInvalidSignature
	}
	return newNotification(fields, true)
}

// ParseUnverified parses a callback without a signature check, for merchant
// profiles that send none. Anyone can forge such a request, so use it only to
// decide which transaction to confirm with Checkout.VerifyApproval, never as
// payment evidence.
func (s *CallbackService) ParseUnverified(body []byte) (*Notification, error) {
	fields, err := callbackFields(body)
	if err != nil {
		return nil, err
	}
	return newNotification(fields, false)
}

// callbackFields parses a bounded JSON object, rejecting duplicate keys and
// trailing data.
func callbackFields(body []byte) (map[string]json.RawMessage, error) {
	if len(body) == 0 || len(body) > MaxCallbackBytes {
		return nil, fmt.Errorf("%w: body must be 1 to %d bytes", ErrInvalidCallback, MaxCallbackBytes)
	}
	d := json.NewDecoder(bytes.NewReader(body))
	if tok, err := d.Token(); err != nil || tok != json.Delim('{') {
		return nil, fmt.Errorf("%w: body is not a JSON object", ErrInvalidCallback)
	}
	fields := map[string]json.RawMessage{}
	for d.More() {
		tok, err := d.Token()
		if err != nil {
			return nil, fmt.Errorf("%w: malformed JSON", ErrInvalidCallback)
		}
		key := tok.(string) // object keys are always strings
		if _, dup := fields[key]; dup {
			return nil, fmt.Errorf("%w: duplicate field %q", ErrInvalidCallback, key)
		}
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return nil, fmt.Errorf("%w: malformed JSON", ErrInvalidCallback)
		}
		fields[key] = value
	}
	if _, err := d.Token(); err != nil {
		return nil, fmt.Errorf("%w: malformed JSON", ErrInvalidCallback)
	}
	if _, err := d.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: data after the JSON object", ErrInvalidCallback)
	}
	return fields, nil
}

// signedCallbackString concatenates field values in key order, as PayWay signs them.
func signedCallbackString(fields map[string]json.RawMessage) (string, error) {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out strings.Builder
	for _, k := range keys {
		v, err := phpString(fields[k])
		if err != nil {
			return "", fmt.Errorf("%w: field %q: %v", ErrInvalidCallback, k, err)
		}
		out.WriteString(v)
	}
	return out.String(), nil
}

// phpString converts a JSON value the way PHP's string conversion does.
func phpString(raw json.RawMessage) (string, error) {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 {
		return "", errors.New("empty value")
	}
	switch b[0] {
	case '"':
		var s string
		err := json.Unmarshal(b, &s)
		return s, err
	case '{', '[':
		var c bytes.Buffer
		err := json.Compact(&c, b)
		return c.String(), err
	}
	switch string(b) {
	case "null", "false":
		return "", nil
	case "true":
		return "1", nil
	}
	// PHP decodes an integer literal to int and prints it unchanged.
	if !bytes.ContainsAny(b, ".eE") {
		return string(b), nil
	}
	f, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return "", errors.New("invalid number")
	}
	// PHP prints whole floats below 1e14 without an exponent or decimal
	// point, so 2000.0 signs as "2000" and 2000000.0 as "2000000".
	if f == math.Trunc(f) && math.Abs(f) < 1e14 {
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	}
	return strconv.FormatFloat(f, 'g', -1, 64), nil
}

func newNotification(fields map[string]json.RawMessage, verified bool) (*Notification, error) {
	text := func(k string) string {
		if v, ok := fields[k]; ok {
			s, _ := phpString(v)
			return s
		}
		return ""
	}
	amount := func(k string) (Amount, error) {
		var a Amount
		if v, ok := fields[k]; ok {
			if err := a.UnmarshalJSON(v); err != nil {
				return "", fmt.Errorf("%w: field %q: %v", ErrInvalidCallback, k, err)
			}
		}
		return a, nil
	}
	n := &Notification{
		TranID:           text("tran_id"),
		Status:           text("status"),
		APV:              text("apv"),
		BankRef:          text("bank_ref"),
		PaymentType:      text("payment_type"),
		OriginalCurrency: text("original_currency"),
		PaymentCurrency:  text("payment_currency"),
		TransactionDate:  text("transaction_date"),
		ReturnParams:     text("return_params"),
		PayerAccount:     text("payer_account"),
		BankName:         text("bank_name"),
		CardSource:       text("card_source"),
		Verified:         verified,
		Fields:           fields,
	}
	if err := validateTranID(n.TranID); err != nil {
		return nil, fmt.Errorf("%w: tran_id must be 1 to 20 characters", ErrInvalidCallback)
	}
	var err error
	for _, f := range []struct {
		key string
		dst *Amount
	}{
		{"original_amount", &n.OriginalAmount},
		{"payment_amount", &n.PaymentAmount},
		{"total_amount", &n.TotalAmount},
		{"discount_amount", &n.DiscountAmount},
	} {
		if *f.dst, err = amount(f.key); err != nil {
			return nil, err
		}
	}
	return n, nil
}
