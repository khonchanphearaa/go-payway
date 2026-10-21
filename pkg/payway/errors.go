package payway

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Sentinel errors for errors.Is. PayWay reuses status codes with different
// meanings per endpoint (code 6 is "not found" for check-transaction but
// "domain not whitelisted" for generate-qr), so match these instead of
// comparing Error.Code.
var (
	ErrTransactionNotFound  = errors.New("payway: transaction not found")
	ErrInvalidHash          = errors.New("payway: invalid request hash")
	ErrRateLimited          = errors.New("payway: request limit reached")
	ErrNotWhitelisted       = errors.New("payway: domain or URL not whitelisted")
	ErrDuplicateTransaction = errors.New("payway: duplicate transaction ID")

	// ErrInconsistentTransaction means PayWay's own responses disagree about a
	// payment, so it must not be treated as approved.
	ErrInconsistentTransaction = errors.New("payway: inconsistent transaction state")

	// ErrInvalidSignature means a callback's HMAC does not match its body.
	ErrInvalidSignature = errors.New("payway: invalid callback signature")
	// ErrInvalidCallback means a callback body is not a well-formed PayWay notification.
	ErrInvalidCallback = errors.New("payway: invalid callback payload")
)

// endpointCodes maps each endpoint's documented status codes to sentinels.
var endpointCodes = map[string]map[string]error{
	pathGenerateQR: {
		"1": ErrInvalidHash, "6": ErrNotWhitelisted, "102": ErrNotWhitelisted,
		"403": ErrDuplicateTransaction, "429": ErrRateLimited,
	},
	pathCheckTransaction:   {"5": ErrInvalidHash, "6": ErrTransactionNotFound, "429": ErrRateLimited},
	pathTransactionDetails: {"5": ErrInvalidHash, "6": ErrTransactionNotFound, "429": ErrRateLimited},
	pathCloseTransaction:   {"1": ErrInvalidHash, "5": ErrTransactionNotFound},
}

// Error is a structured error returned by the SDK. It carries PayWay's status
// code, message and trace ID, never the raw response body. Use errors.Is with
// the sentinels above, or errors.As to read the fields.
type Error struct {
	// HTTPStatus is set when PayWay answered with a non-2xx HTTP status.
	HTTPStatus int
	Code       string
	Message    string
	TraceID    string

	path string // endpoint, used to interpret Code
}

func (e *Error) Error() string {
	if e.HTTPStatus != 0 {
		return fmt.Sprintf("payway error [http %d, code %s]: %s (trace: %s)", e.HTTPStatus, e.Code, e.Message, e.TraceID)
	}
	return fmt.Sprintf("payway error [%s]: %s (trace: %s)", e.Code, e.Message, e.TraceID)
}

// Is matches e against the sentinel errors using the endpoint it came from.
func (e *Error) Is(target error) bool {
	if target == nil {
		return false
	}
	if target == ErrRateLimited && e.HTTPStatus == http.StatusTooManyRequests {
		return true
	}
	return endpointCodes[e.path][normalizeCode(e.Code)] == target
}

// statusError builds the error for a PayWay status envelope that is not a success.
func statusError(path string, s APIStatus) *Error {
	return &Error{Code: s.Code, Message: s.Message, TraceID: s.TraceID, path: path}
}

// normalizeCode strips leading zeros from numeric codes, so "06" matches "6".
func normalizeCode(code string) string {
	if code == "" || strings.Trim(code, "0123456789") != "" {
		return code
	}
	if trimmed := strings.TrimLeft(code, "0"); trimmed != "" {
		return trimmed
	}
	return "0"
}
