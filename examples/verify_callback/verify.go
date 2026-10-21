package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/khonchanphearaa/go-payway/pkg/payway"
)

// A callback only says "something happened to this transaction". This handler
// verifies its signature, acknowledges it quickly, and then asks PayWay's
// authenticated APIs whether the payment really completed before fulfilling.
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
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		n, err := client.Callback.VerifyRequest(r)
		if err != nil {
			// SDK errors never contain the body or signature.
			log.Printf("callback rejected: %v", err)
			http.Error(w, "invalid callback", http.StatusUnauthorized)
			return
		}

		// Store the event idempotently first (for example keyed by a SHA-256
		// of the body), so a retried callback is not processed twice.
		w.WriteHeader(http.StatusOK)

		// Confirm out of band; in production, hand this to a durable worker.
		go confirm(client, n.TranID)
	})

	log.Println("Listening on :8080")
	srv := &http.Server{Addr: ":8080", ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(srv.ListenAndServe())
}

func confirm(client *payway.Client, tranID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	v, err := client.Checkout.VerifyApproval(ctx, tranID)
	switch {
	case errors.Is(err, payway.ErrRateLimited):
		log.Printf("%s: rate limited, retry later", tranID)
	case err != nil:
		log.Printf("%s: not confirmed, retry later: %v", tranID, err)
	case !v.Approved:
		log.Printf("%s: not paid (status %s)", tranID, v.PaymentStatus)
	default:
		// Compare with the order you stored before fulfilling, e.g.
		// v.OriginalAmount.Equal(order.Amount) && v.OriginalCurrency == order.Currency.
		log.Printf("%s: paid %s %s (apv %s)", tranID, v.OriginalAmount, v.OriginalCurrency, v.APV)
	}
}
