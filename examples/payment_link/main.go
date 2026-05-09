package main

import (
	"context"
	"log"
	"fmt"

	"github.com/khonchanphearaa/go-payway/pkg/payway"
)

func main() {

	client, err := payway.NewClient(payway.Config{
		MerchantID: "eroxisabaypaygoods",
		APIKey:     "22e9e0cf-d5b4-4a31-82db-bc1046brewefwf",
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
