# SDK KHQR Go-Payway

A clean, well-documented Go SDK for **ABA PayWay** — built for Cambodian developers

No more copy-pasting HMAC-SHA512 logic. No more manually Base64-encoding items arrays
Just a simple, typed API that handles all the complexity for you.

---

## Go SDK Feature

- QR code generation (ABA PAY + KHQR)
- Ecommerce checkout (purchase, check, close, refund)
- Payment link creation
- Callback verification (HMAC-SHA512)
- Automatic hash generation for every request
- Automatic Base64 encoding for items, URLs
- Sandbox + Production support
- Pure Go — zero external dependencies

---

## Installation

```bash
go get github.com/khonchanphearaa/go-payway
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
    PaymentOption: payway.PaymentOptionABAPayUSD,
    FirstName:     "Khon",
    LastName:      "Chanpheara",
    Email:         "phearaa@example.com",
    Phone:         "012345678",
    CallbackURL:   "https://yourshop.com/webhook/payway",
    Items: []encoder.Item{
        {Name: "Coffee", Quantity: 2, Price: 2.50},
    },
})
```

### 3. Verify a Callback

```go
http.HandleFunc("/webhook/payway", func(w http.ResponseWriter, r *http.Request) {
    payload, err := client.Callback.ParseAndVerify(r)
    if err != nil {
        http.Error(w, "invalid callback", http.StatusBadRequest)
        return
    }
    if payload.IsPaid() {
        fmt.Printf("Paid: TranID=%s Amount=%.2f %s\n",
            payload.TranID, payload.Amount, payload.Currency)
    }

    w.WriteHeader(http.StatusOK)
})
```

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
| `GetDetails(ctx, *TransactionDetailsRequest)` | Get full transaction details |
| `CheckTransaction(ctx, tranID)` | Lightweight payment status poll |
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
| `ParseAndVerify(r *http.Request)` | Parse + verify a callback HTTP request |
| `Verify(*CallbackPayload)` | Verify a pre-parsed payload |

---

## Constants

```go
// Currency
payway.CurrencyUSD
payway.CurrencyKHR

// Payment options
payway.PaymentOptionABAPay       
payway.PaymentOptionKHQR         
payway.PaymentOptionABAPayKHQR   
payway.PaymentOptionAll          // + WeChat + Alipay (USD only)

// QR templates
payway.QRTemplateDefault
payway.QRTemplateColor
payway.QRTemplateDark
```

## Running Tests

```bash
go test ./...
```

## Running Generate QRCode

This can testing generate qrcode example

```bash
go run ./example/qr/generatekhqr.go
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

PRs and issues are welcome! This library is built to help the Cambodian developer community.

---
