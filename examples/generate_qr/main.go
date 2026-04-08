package main

import (
	"context"
	"fmt"
	"log"

	"github.com/khonchanphearaa/go-payway/pkg/encoder" 			// encoder itmes
	"github.com/khonchanphearaa/go-payway/pkg/payway"
)

func main() {
	client, err := payway.NewClient(payway.Config{
		MerchantID: "eroxisabaypaygoods",
		APIKey:     "22e9e0cf-d5b4-4a31-82db-bc1046brewefwf",
		Sandbox: true,   // false for production
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.QR.Generate(context.Background(), &payway.QRRequest{
		TransactionID: "TXN-20250404-001",
		Amount:        5.00,
		Currency:      payway.CurrencyUSD,    // Supported KHR, USD
		PaymentOption: payway.PaymentOptionABAPayKHQR,
		FirstName:     "Khon",
		LastName:      "Chanpheara",
		Email:         "phearaa@example.com",
		Phone:         "012345678",
		CallbackURL:   "https://yourshop.com/webhook/payway",
		Items: []encoder.Item{
			{Name: "Coffee", Quantity: 2, Price: 2.50},  // This items can be optional
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("QR String: %s\n", resp.QRString)
	fmt.Printf("QR Image (data URI): %s\n", resp.QRImage)
	fmt.Printf("ABAPay Deeplink: %s\n", resp.ABAPayDeeplink)
	fmt.Printf("App Store URL: %s\n", resp.AppStore)
	fmt.Printf("Play Store URL: %s\n", resp.PlayStore)
}
