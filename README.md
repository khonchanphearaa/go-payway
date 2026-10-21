# SDK KHQR Go-Payway

A clean, well-documented Go SDK for **ABA PayWay** — built for Cambodian developers

No more copy-pasting HMAC-SHA512 logic. No more manually Base64-encoding items arrays
Just a simple, typed API that handles all the complexity for you.

---

## Go SDK Feature

- QR code generation (ABA PAY + KHQR)
- Ecommerce checkout (purchase, check, close, refund)
- Payment link creation
- Payment confirmation: `VerifyApproval` cross-checks check-transaction and transaction-detail
- Callback verification (`X-PayWay-HMAC-SHA512`, constant-time)
- Exact amounts (`payway.Amount`), never rounded floats
- Typed errors per endpoint (`errors.Is(err, payway.ErrTransactionNotFound)`)
- Safe HTTP defaults: no redirects, 10s timeout, bounded responses
- Automatic hash generation for every request
- Automatic Base64 encoding for items, URLs
- Sandbox + Production support
- Pure Go — zero external dependencies

---

## Installation

```bash
go get github.com/khonchanphearaa/go-payway
```

OR upgrade to last versions

```bash
go get github.com/khonchanphearaa/go-payway@latest
```

---

## Quick Start

### 1. Create a client

```go
import "github.com/khonchanphearaa/go-payway/pkg/payway"

client, err := payway.NewClient(payway.Config{
    MerchantID: os.Getenv("PAYWAY_MERCHANT_ID"),
    APIKey:     os.Getenv("PAYWAY_API_KEY"),
    Sandbox:    true, // false for production
})
```

### 2. Generate a QR Code

```go
import "github.com/khonchanphearaa/go-payway/pkg/encoder"

resp, err := client.QR.Generate(ctx, &payway.QRRequest{
    TransactionID: "TXN-20250404-001",
    Amount:        5.00,
    Currency:      payway.CurrencyUSD,
    PaymentOption: payway.PaymentOptionABAPayKHQR, // the QR API accepts abapay_khqr, wechat or alipay
    FirstName:     "Khon",
    LastName:      "Chanpheara",
    Email:         "phearaa@example.com",
    Phone:         "012345678",
    CallbackURL:   "https://yourshop.com/webhook/payway",

    // Can be optinal provide
    Items: []encoder.Item{      
        {Name: "Coffee", Quantity: 2, Price: 2.50},
    },
})
```

### 3. Handle the callback, then confirm the payment

A callback tells you to check a transaction; it is **not** proof of payment.
Verify its signature, acknowledge it, then confirm with `VerifyApproval`
before you fulfil the order.

```go
http.HandleFunc("/webhook/payway", func(w http.ResponseWriter, r *http.Request) {
    n, err := client.Callback.VerifyRequest(r) // reads X-PayWay-HMAC-SHA512, max 8 KiB
    if err != nil {
        http.Error(w, "invalid callback", http.StatusUnauthorized)
        return
    }
    // Store the event idempotently, then acknowledge quickly.
    w.WriteHeader(http.StatusOK)
    go confirm(n.TranID) // in production: a durable worker with retries
})

func confirm(tranID string) {
    v, err := client.Checkout.VerifyApproval(ctx, tranID)
    if err != nil {
        return // retry later; errors.Is(err, payway.ErrRateLimited) etc.
    }
    if v.Approved && v.OriginalAmount.Equal(order.Amount) && v.OriginalCurrency == order.Currency {
        // fulfil the order exactly once
    }
}
```

`VerifyApproval` polls check-transaction and, only for an approval, confirms it
with transaction-detail (limited by PayWay to 10 requests per minute). Both must
agree on status and exact amount, and details must name your `tran_id` and its
original currency, otherwise the error wraps `ErrInconsistentTransaction`.
Status code 0 alone also covers `PRE-AUTH`, so it is not treated as paid.

If your merchant profile sends callbacks without a signature, use
`client.Callback.ParseUnverified(body)` only to learn which `tran_id` to confirm.

---

## API Reference

### `client.QR`

| Method | Description |
|---|---|
| `Generate(ctx, *QRRequest)` | Generate a QR code for payment |

### `client.Checkout`

| Method | Description |
|---|---|
| `Purchase(ctx, *PurchaseRequest)` | Initiate a hosted checkout session |
| `GetDetails(ctx, *TransactionDetailsRequest)` | Get full transaction details (10 requests/minute) |
| `VerifyApproval(ctx, tranID)` | Decide whether a payment completed (check + details) |
| `CheckTransaction(ctx, tranID)` | Lightweight payment status poll (last 7 days) |
| `CloseTransaction(ctx, tranID)` | Cancel an active transaction |
| `Refund(ctx, *RefundRequest)` | Refund a completed transaction |
| `GetExchangeRate(ctx)` | Get current USD/KHR rate |

### `client.PaymentLink`

| Method | Description |
|---|---|
| `Create(ctx, *CreatePaymentLinkRequest)` | Create a shareable payment link |
| `GetDetails(ctx, tranID)` | Get payment link status |

### `client.Callback`

| Method | Description |
|---|---|
| `VerifyRequest(r *http.Request)` | Read and verify a callback request |
| `VerifyBody(body, signature)` | Verify a raw body against its signature |
| `ParseUnverified(body)` | Parse an unsigned callback (never payment evidence) |
| `ParseAndVerify`, `Verify` | Deprecated: legacy payload shape |

---

## Constants

```go
// Currency
payway.CurrencyUSD
payway.CurrencyKHR

// Payment options
payway.PaymentOptionABAPayKHQR   // QR API: ABA KHQR (KHR or USD)
payway.PaymentOptionWeChat       // QR API: USD only
payway.PaymentOptionAlipay       // QR API: USD only
payway.PaymentOptionABAPay       // checkout
payway.PaymentOptionKHQR         // checkout only; the QR API rejects it (code 04)
payway.PaymentOptionAll          // checkout: + WeChat + Alipay (USD only)

// Status codes
payway.CodeInvalidData           // "04": a request field failed validation
payway.CodeTransactionNotFound   // "6": PayWay has no record of the tran_id

// Payment statuses (Transaction.PaymentStatus)
payway.StatusApproved, payway.StatusPreAuth, payway.StatusPending
payway.StatusDeclined, payway.StatusRefunded, payway.StatusCancelled

// QR templates
payway.QRTemplateDefault
payway.QRTemplateColor
payway.QRTemplateDark
payway.QRTemplateDigital
```

## Errors

PayWay rejections are returned as `*payway.Error` with PayWay's status code,
message and trace ID, never the raw response body. PayWay reuses codes with
different meanings per endpoint (code 6 is "not found" for check-transaction
but "domain not whitelisted" for generate-qr), so match sentinels with
`errors.Is` rather than comparing `Code`:

```go
switch {
case errors.Is(err, payway.ErrTransactionNotFound):     // check 6, details 6, close 5
case errors.Is(err, payway.ErrRateLimited):             // code or HTTP 429: retry later
case errors.Is(err, payway.ErrInvalidHash):             // wrong API key or merchant ID
case errors.Is(err, payway.ErrNotWhitelisted):          // register your domain with PayWay
case errors.Is(err, payway.ErrDuplicateTransaction):    // tran_id already used
case errors.Is(err, payway.ErrInconsistentTransaction): // do not fulfil; investigate
}
var perr *payway.Error
if errors.As(err, &perr) {
    log.Printf("http=%d code=%s trace=%s", perr.HTTPStatus, perr.Code, perr.TraceID)
}
```

Pass `Config.HTTPClient` to supply your own `*http.Client` (transport, proxy,
tests); the SDK copies it. `Config.HTTPTimeout` overrides its timeout (default
10s). Redirects are refused unless your client sets its own `CheckRedirect`,
because following one would resend the signed request to another host.
Responses larger than 2 MiB are rejected.

## Exact amounts

`payway.Amount` keeps the decimal text PayWay sent. Compare it with
`Equal` ("2000" equals "2000.00") or convert with `Rat`; never parse it as a
float. Check and details responses expose it under `.Transaction`; the older
float fields in `.Data` remain for compatibility and are deprecated.

## Running Tests

```bash
go test ./...
```

## Running Generate QRCode

This can testing generate qrcode example

```bash
PAYWAY_MERCHANT_ID=... PAYWAY_API_KEY=... go run ./examples/generate_qr
```

---

## Sandbox vs Production

| | Sandbox | Production |
|---|---|---|
| Base URL | `checkout-sandbox.payway.com.kh` | `checkout.payway.com.kh` |
| Config | `Sandbox: true` | `Sandbox: false` |
| Register | [sandbox.payway.com.kh](https://sandbox.payway.com.kh/register-sandbox/) | Contact PayWay sales |

---

## Contributing

PRs and issues are welcome!

---
