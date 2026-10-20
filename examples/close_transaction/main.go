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

	resp, err := client.Checkout.CloseTransaction(context.Background(), "TXN-20250404-001")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Status code: %s\n", resp.Status.Code)
	fmt.Printf("Status message: %s\n", resp.Status.Message)
}
