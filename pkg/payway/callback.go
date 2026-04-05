package payway

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/khonchanphearaa/go-payway/pkg/hash"
)

type CallbackService struct {
	hasher *hash.Generator
}

type CallbackPayload struct {
	TranID        string  `json:"tran_id"`
	Status        string  `json:"status"` // "0" = success
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	PaymentOption string  `json:"payment_option"`
	MerchantID    string  `json:"merchant_id"`
	FirstName     string  `json:"firstname"`
	LastName      string  `json:"lastname"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	CustomFields  string  `json:"custom_fields"`
	ReturnParams  string  `json:"return_params"`
	Hash          string  `json:"hash"`
}

func (p *CallbackPayload) IsPaid() bool {
	return p.Status == "0"
}

func (s *CallbackService) ParseAndVerify(r *http.Request) (*CallbackPayload, error) {
	var payload CallbackPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("payway/callback: failed to parse request body: %w", err)
	}

	if err := s.Verify(&payload); err != nil {
		return nil, err
	}

	return &payload, nil
}

func (s *CallbackService) Verify(payload *CallbackPayload) error {
	params := map[string]string{
		"tran_id":        payload.TranID,
		"status":         payload.Status,
		"amount":         fmt.Sprintf("%.2f", payload.Amount),
		"currency":       payload.Currency,
		"payment_option": payload.PaymentOption,
		"merchant_id":    payload.MerchantID,
		"firstname":      payload.FirstName,
		"lastname":       payload.LastName,
		"email":          payload.Email,
		"phone":          payload.Phone,
		"custom_fields":  payload.CustomFields,
		"return_params":  payload.ReturnParams,
	}

	valid, err := s.hasher.Verify(params, payload.Hash)
	if err != nil {
		return fmt.Errorf("payway/callback: hash verification error: %w", err)
	}
	if !valid {
		return fmt.Errorf("payway/callback: hash mismatch — possible tampered callback")
	}

	return nil
}
