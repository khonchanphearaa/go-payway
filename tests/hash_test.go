package encoder_test

import (
	"testing"

	"github.com/khonchanpharaa/go-payway/pkg/hash"
)

func TestGenerate_DeterministicOutput(t *testing.T) {
	g := hash.New("test-secret-key")
	params := map[string]string{
		"merchant_id": "my-merchant",
		"tran_id":     "TXN-001",
		"amount":      "10.00",
		"currency":    "USD",
	}
	h1, err := g.Generate(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h2, err := g.Generate(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 != h2 {
		t.Error("same params should produce same hash")
	}
}

func TestGenerate_DifferentKeysProduceDifferentHashes(t *testing.T) {
	params := map[string]string{"tran_id": "TXN-001"}
	h1, _ := hash.New("key-a").Generate(params)
	h2, _ := hash.New("key-b").Generate(params)
	if h1 == h2 {
		t.Error("different API keys should produce different hashes")
	}
}

func TestGenerate_EmptyKeyReturnsError(t *testing.T) {
	g := hash.New("")
	_, err := g.Generate(map[string]string{"tran_id": "x"})
	if err == nil {
		t.Error("expected error for empty API key")
	}
}

func TestGenerate_EmptyFieldsAreExcluded(t *testing.T) {
	g := hash.New("secret")
	// Hash with empty field should equal hash without that field at all
	withEmpty := map[string]string{"tran_id": "TXN-001", "email": ""}
	withoutEmpty := map[string]string{"tran_id": "TXN-001"}
	h1, _ := g.Generate(withEmpty)
	h2, _ := g.Generate(withoutEmpty)
	if h1 != h2 {
		t.Error("empty fields should be excluded from hash — results should match")
	}
}

func TestVerify_ValidHash(t *testing.T) {
	g := hash.New("secret")
	params := map[string]string{"merchant_id": "test", "amount": "25.00"}
	generated, err := g.Generate(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	valid, err := g.Verify(params, generated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("expected hash to be valid")
	}
}

func TestVerify_TamperedHash(t *testing.T) {
	g := hash.New("secret")
	params := map[string]string{"merchant_id": "test"}
	valid, err := g.Verify(params, "tampered-hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("tampered hash should be invalid")
	}
}

func TestGenerateOrdered_OrderAffectsOutput(t *testing.T) {
	g := hash.New("secret")
	h1, err := g.GenerateOrdered("A", "B", "C")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	h2, err := g.GenerateOrdered("A", "C", "B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h1 == h2 {
		t.Error("different order should produce different hash")
	}
}
