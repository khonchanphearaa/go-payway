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

	resp, err := client.Checkout.GetDetails(context.Background(), &payway.TransactionDetailsRequest{
		TransactionID: "TXN-20250404-001",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Transaction ID: %s\n", resp.Data.TransactionID)
	fmt.Printf("Amount: %f\n", resp.Amount)
	fmt.Printf("Currency: %s\n", resp.Currency)
	//... other fields
}
