
package encoder

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

func ToBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func FromBase64(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("payway/encoder: failed to decode base64: %w", err)
	}
	return string(b), nil
}

type Item struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

func EncodeItems(items []Item) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("payway/encoder: at least one item is required")
	}

	raw, err := json.Marshal(items)
	if err != nil {
		return "", fmt.Errorf("payway/encoder: failed to marshal items: %w", err)
	}
	return ToBase64(string(raw)), nil
}

func DecodeItems(encoded string) ([]Item, error) {
	decoded, err := FromBase64(encoded)
	if err != nil {
		return nil, err
	}

	var items []Item
	if err := json.Unmarshal([]byte(decoded), &items); err != nil {
		return nil, fmt.Errorf("payway/encoder: failed to unmarshal items: %w", err)
	}
	return items, nil
}
