package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
		log.Fatalf("failed to create PayWay client: %v", err)
	}

	http.HandleFunc("/webhook/payway", func(w http.ResponseWriter, r *http.Request) {
		payload, err := client.Callback.ParseAndVerify(r)
		if err != nil {
			http.Error(w, "invalid callback", http.StatusBadRequest)
			log.Printf("Callback rejected: %v", err)
			return
		}

		// Check payment status
		if payload.IsPaid() {
			log.Printf("Payment received! TranID=%s Amount=%.2f %s",
				payload.TranID, payload.Amount, payload.Currency)
		} else {
			log.Printf("Transaction %s status: %s", payload.TranID, payload.Status)
		}

		// Response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"received": "ok"})
	})

	fmt.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
