package payway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
)

// Amount is a decimal amount exactly as PayWay sent it, for example "2000" or
// "0.01". Unlike float64 it never rounds, so it can be compared with the amount
// a merchant charged. Use Rat or Equal; do not parse it as a float.
type Amount string

// amountPattern accepts plain decimals only. Exponents are rejected so a
// hostile value cannot make big.Rat allocate an enormous number.
var amountPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]{0,17})(\.[0-9]{1,18})?$`)

// UnmarshalJSON accepts a JSON number, a numeric string or null. The text is
// stored as received; Valid reports whether it is a usable decimal.
func (a *Amount) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	switch {
	case bytes.Equal(b, []byte("null")):
		*a = ""
	case len(b) > 0 && b[0] == '"':
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*a = Amount(s)
	case len(b) > 0 && (b[0] == '-' || (b[0] >= '0' && b[0] <= '9')):
		*a = Amount(b)
	default:
		return fmt.Errorf("payway: amount must be a number or numeric string")
	}
	return nil
}

// Valid reports whether a is a plain decimal within PayWay's range.
func (a Amount) Valid() bool {
	return amountPattern.MatchString(string(a))
}

// Rat returns the exact value of a, or false when a is empty or invalid.
func (a Amount) Rat() (*big.Rat, bool) {
	if !a.Valid() {
		return nil, false
	}
	return new(big.Rat).SetString(string(a))
}

// Equal reports whether a and b are valid and numerically equal, so "2000"
// equals "2000.00".
func (a Amount) Equal(b Amount) bool {
	x, ok := a.Rat()
	if !ok {
		return false
	}
	y, ok := b.Rat()
	return ok && x.Cmp(y) == 0
}

func (a Amount) String() string { return string(a) }
