package hash

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
)

type Generator struct {
	apiKey string
}

func New(apiKey string) *Generator {
	return &Generator{apiKey: apiKey}
}

func (g *Generator) Generate(params map[string]string) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("payway: api key must not be empty")
	}
	raw := buildSortedString(params)

	mac := hmac.New(sha512.New, []byte(g.apiKey))
	if _, err := mac.Write([]byte(raw)); err != nil {
		return "", fmt.Errorf("payway: failed to compute hash: %w", err)
	}

	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (g *Generator) GenerateOrdered(values ...string) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("payway: api key must not be empty")
	}

	raw := strings.Join(values, "")

	mac := hmac.New(sha512.New, []byte(g.apiKey))
	if _, err := mac.Write([]byte(raw)); err != nil {
		return "", fmt.Errorf("payway: failed to compute hash: %w", err)
	}

	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (g *Generator) Verify(params map[string]string, receivedHash string) (bool, error) {
	expected, err := g.Generate(params)
	if err != nil {
		return false, err
	}
	return hmac.Equal([]byte(expected), []byte(receivedHash)), nil
}

// Equal reports whether signature is the Base64 HMAC-SHA512 of data. It
// compares the decoded bytes in constant time and rejects an empty key or a
// signature longer than a SHA-512 MAC can encode to.
func (g *Generator) Equal(data []byte, signature string) bool {
	if g.apiKey == "" || signature == "" || len(signature) > base64.StdEncoding.EncodedLen(sha512.Size) {
		return false
	}
	got, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha512.New, []byte(g.apiKey))
	mac.Write(data)
	return hmac.Equal(got, mac.Sum(nil))
}

func buildSortedString(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		if params[k] != "" {
			sb.WriteString(params[k])
		}
	}
	return sb.String()
}
