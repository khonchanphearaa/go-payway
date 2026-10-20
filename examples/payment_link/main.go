package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/khonchanphearaa/go-payway/pkg/payway"
)

func main() {

	client, err := payway.NewClient(payway.Config{
		MerchantID: os.Getenv("PAYWAY_MERCHANT_ID"),
		APIKey:     os.Getenv("PAYWAY_API_KEY"),
		Sandbox:    true,
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.PaymentLink.Create(context.Background(), &payway.CreatePaymentLinkRequest{
		MerchantAuth: "RSA PUBLIC KEY PROVIDED BY ABA BANK",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Payment Link: %s\n", resp.PaymentLink)
}
