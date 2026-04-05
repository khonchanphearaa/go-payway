package encoder_test

import (
	"testing"

	"github.com/khonchanpharaa/go-payway/pkg/encoder"
)

func TestEncodeDecodeItems_RoundTrip(t *testing.T) {
	items := []encoder.Item{
		{Name: "Phone Case", Quantity: 1, Price: 9.99},
		{Name: "Screen Protector", Quantity: 2, Price: 3.50},
	}

	encoded, err := encoder.EncodeItems(items)
	if err != nil {
		t.Fatalf("EncodeItems failed: %v", err)
	}
	if encoded == "" {
		t.Fatal("expected non-empty encoded string")
	}

	decoded, err := encoder.DecodeItems(encoded)
	if err != nil {
		t.Fatalf("DecodeItems failed: %v", err)
	}

	if len(decoded) != len(items) {
		t.Fatalf("expected %d items, got %d", len(items), len(decoded))
	}
	for i, item := range items {
		if decoded[i].Name != item.Name {
			t.Errorf("item[%d] Name: got %q, want %q", i, decoded[i].Name, item.Name)
		}
		if decoded[i].Price != item.Price {
			t.Errorf("item[%d] Price: got %v, want %v", i, decoded[i].Price, item.Price)
		}
	}
}

func TestEncodeItems_EmptyReturnsError(t *testing.T) {
	_, err := encoder.EncodeItems(nil)
	if err == nil {
		t.Error("expected error for empty items, got nil")
	}
}

func TestToBase64_FromBase64_RoundTrip(t *testing.T) {
	original := "https://myshop.com/callback"

	encoded := encoder.ToBase64(original)
	decoded, err := encoder.FromBase64(encoded)
	if err != nil {
		t.Fatalf("FromBase64 failed: %v", err)
	}
	if decoded != original {
		t.Errorf("round-trip failed: got %q, want %q", decoded, original)
	}
}

func TestFromBase64_InvalidInput(t *testing.T) {
	_, err := encoder.FromBase64("!!!not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}
