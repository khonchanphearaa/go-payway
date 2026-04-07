package main

import (
	"context"
	"fmt"
	"log"

	"github.com/khonchanphearaa/go-payway/pkg/payway"
)

func main() {
	client, err := payway.NewClient(payway.Config{
		MerchantID: "eroxisabaypaygoods",
		APIKey:     "22e9e0cf-d5b4-4a31-82db-bc1046brewefwf",
		Sandbox: true,  // false for production
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Checkout.CheckTransaction(context.Background(), "TXN-20250404-001")
	if err != nil{
		log.Fatal(err)
	}
	fmt.Printf("Status: %s\n", resp.Data.PaymentStatus)
	fmt.Printf("Transaction Date: %s\n", resp.Data.TransactionDate) 
	//...
}