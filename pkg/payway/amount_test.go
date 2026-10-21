package payway_test

import (
	"encoding/json"
	"testing"

	"github.com/khonchanphearaa/go-payway/pkg/payway"
)

func TestAmountDecodesExactly(t *testing.T) {
	for in, want := range map[string]payway.Amount{
		`2000`:              "2000",
		`2000.0000000001`:   "2000.0000000001",
		`"0.01"`:            "0.01",
		`null`:              "",
		`123456789012.3456`: "123456789012.3456",
	} {
		var a payway.Amount
		if err := json.Unmarshal([]byte(in), &a); err != nil || a != want {
			t.Errorf("%s: got %q %v", in, a, err)
		}
	}
	for _, in := range []string{`true`, `{}`, `[1]`} {
		var a payway.Amount
		if err := json.Unmarshal([]byte(in), &a); err == nil {
			t.Errorf("accepted %s", in)
		}
	}
}

func TestAmountValidityAndEquality(t *testing.T) {
	for _, a := range []payway.Amount{"", "2e3", "1e999999999", "01", "1.", ".5", "NaN", "1 ", "12345678901234567890"} {
		if a.Valid() {
			t.Errorf("valid %q", a)
		}
		if _, ok := a.Rat(); ok {
			t.Errorf("rat %q", a)
		}
	}
	for _, pair := range [][2]payway.Amount{{"2000", "2000.00"}, {"0.01", "0.010"}, {"-1", "-1.0"}} {
		if !pair[0].Equal(pair[1]) {
			t.Errorf("%q != %q", pair[0], pair[1])
		}
	}
	for _, pair := range [][2]payway.Amount{{"2000", "2000.0000000001"}, {"0.1", "0.01"}, {"", ""}, {"2e3", "2000"}} {
		if pair[0].Equal(pair[1]) {
			t.Errorf("%q == %q", pair[0], pair[1])
		}
	}
}
