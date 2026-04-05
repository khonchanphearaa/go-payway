package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/khonchanphearaa/go-payway/pkg/encoder"
	"github.com/khonchanphearaa/go-payway/pkg/payway"
)

func main() {
	_ = godotenv.Overload()

	merchantID := strings.TrimSpace(os.Getenv("PAYWAY_MERCHANT_ID"))
	apiKey := strings.TrimSpace(os.Getenv("PAYWAY_API_KEY"))
	if merchantID == "" || apiKey == "" {
		log.Fatal("missing PAYWAY_MERCHANT_ID or PAYWAY_API_KEY")
	}
	if merchantID == "your_merchant_id" || apiKey == "your_api_key" {
		log.Fatal("placeholder credentials detected in environment; set real PayWay sandbox MerchantID and APIKey")
	}

	sandbox := true
	if v := strings.TrimSpace(os.Getenv("PAYWAY_SANDBOX")); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err != nil {
			log.Fatalf("invalid PAYWAY_SANDBOX value %q (use true/false)", v)
		}
		sandbox = parsed
	}

	log.Printf("PayWay config: sandbox=%t merchant_id=%q api_key_len=%d", sandbox, merchantID, len(apiKey))

	// create env
	client, err := payway.NewClient(payway.Config{
		MerchantID: merchantID,
		APIKey:     apiKey,
		Sandbox:    sandbox,
	})
	if err != nil {
		log.Fatalf("failed to create PayWay client: %v", err)
	}

	// generate khqr
	resp, err := client.QR.Generate(context.Background(), &payway.QRRequest{
		TransactionID:   fmt.Sprintf("TXN-%d", time.Now().Unix()),
		Amount:          1000,
		Currency:        payway.CurrencyKHR,
		PaymentOption:   payway.PaymentOptionABAPayKHQR,
		FirstName:       "Dara",
		LastName:        "Chan",
		Email:           "dara@example.com",
		Phone:           "012345678",
		CallbackURL:     "https://yourshop.com/webhook/payway",
		Lifetime:        3, // minutes
		QRImageTemplate: payway.QRTemplateColor,
		Items: []encoder.Item{
			{Name: "Coffee", Quantity: 2, Price: 2.50},
		},
	})
	if err != nil {
		log.Fatalf("generate QR failed: %v", err)
	}

	// response
	fmt.Println("QR Generated!")
	fmt.Printf("QR String:       %s\n", resp.QRString)
	fmt.Printf("ABA Pay Deeplink: %s\n", resp.ABAPayDeeplink)
	fmt.Printf("Amount:          %.2f %s\n", resp.Amount, resp.Currency)

	// resp.QRImage is a Base64 PNG — embed it in <img src="..."> on your frontend
}
