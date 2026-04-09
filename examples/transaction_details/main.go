package main

import (
	"context"
	"fmt"
	"log"

	"github.com/khonchanphearaa/go-payway/pkg/payway"
)

func main(){
	client, err := payway.NewClient(payway.Config{
		MerchantID: "eroxisabaypaygoods",
		APIKey:     "22e9e0cf-d5b4-4a31-82db-bc1046brewefwf",
		Sandbox: true,  // false for production
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Checkout.GetDetails(context.Background(),&payway.TransactionDetailsRequest{
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