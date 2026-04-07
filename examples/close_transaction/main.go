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
		Sandbox: true, 
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