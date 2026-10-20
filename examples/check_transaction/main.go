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
		Sandbox:    true, // false for production
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Checkout.CheckTransaction(context.Background(), "TXN-20250404-001")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Status: %s\n", resp.Data.PaymentStatus)
	fmt.Printf("Transaction Date: %s\n", resp.Data.TransactionDate)
	//...
}
