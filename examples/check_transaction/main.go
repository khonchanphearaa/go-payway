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
	t := resp.Transaction // exact amounts; resp.Data keeps the deprecated floats
	fmt.Printf("Status: %s (approved: %v)\n", t.PaymentStatus, t.Approved())
	fmt.Printf("Original amount: %s\n", t.OriginalAmount)
	fmt.Printf("Transaction Date: %s\n", t.TransactionDate)
	//...
}
