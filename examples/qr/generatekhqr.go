package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	// "github.com/khonchanphearaa/go-payway/pkg/encoder"
	"github.com/khonchanphearaa/go-payway/pkg/payway"
)

func main() {
	godotenv.Load()

	client, err := payway.NewClient(payway.Config{
		MerchantID: os.Getenv("PAYWAY_MERCHANT_ID"),
		APIKey:     os.Getenv("PAYWAY_API_KEY"),
		Sandbox:    true,
	})
	if err != nil {
		log.Fatalf("failed to create PayWay client: %v", err)
	}

	r := gin.Default()

	r.POST("/generatekhqr", func(c *gin.Context) {
		var body struct {
			Amount    float64        `json:"amount"`
			// Items     []encoder.Item `json:"items"`
			FirstName string         `json:"first_name"`
			LastName  string         `json:"last_name"`
			Email     string         `json:"email"`
			Phone     string         `json:"phone"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		resp, err := client.QR.Generate(c.Request.Context(), &payway.QRRequest{
			TransactionID: fmt.Sprintf("TXN-%d", time.Now().UnixMilli()),
			FirstName:     body.FirstName,
			LastName:      body.LastName,
			Amount:        body.Amount,
			Email:         body.Email,
			Phone:         body.Phone,
			Currency:      payway.CurrencyKHR,
			PaymentOption: payway.PaymentOptionABAPayKHQR,
			// Items:         body.Items,
			Lifetime:      10,
		})
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"transaction_id": resp.Status.TraceID,
			"message":        resp.Status.Message,
			"qr_string":      resp.QRString,
			// "qr_image":  resp.QRImage,
			"deeplink": resp.ABAPayDeeplink,
			"amount":   resp.Amount,
			"currency": resp.Currency,
		})
	})

	r.Run(":8080")
}
