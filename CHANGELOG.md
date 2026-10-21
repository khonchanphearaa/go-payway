# Changelog

## v0.5.0 - [2026-10-21]

Additive release: code written for v0.4.0 keeps compiling. Two defaults change
for safety (see "Changed").

### Added

- `Checkout.VerifyApproval(ctx, tranID)`: decides whether a payment completed.
  It polls check-transaction and confirms approvals with transaction-detail,
  requiring matching `transaction_id`, status, exact original amount and
  original currency. A disagreement wraps `ErrInconsistentTransaction`.
- `payway.Amount`: exact decimal amounts with `Valid`, `Rat` and `Equal`.
- `Transaction` (exact view) on `CheckTransactionResponse.Transaction` and
  `TransactionDetailsResponse.Transaction`, including refund and discount
  amounts, `bank_ref`, `payment_type` and `transaction_operations`.
  `Transaction.Approved()` requires status `APPROVED` and status code 0;
  `PreAuthorized()` reports an uncaptured hold.
- Callback verification that matches PayWay's `X-PayWay-HMAC-SHA512` header:
  `Callback.VerifyRequest`, `Callback.VerifyBody` and `Callback.ParseUnverified`,
  returning a `Notification`. All top-level fields are signed in key order,
  formatted as PHP does. Duplicate keys, trailing data and bodies over 8 KiB
  are rejected, and the signature is compared in constant time.
- Sentinel errors for `errors.Is`, mapped per endpoint: `ErrTransactionNotFound`,
  `ErrInvalidHash`, `ErrRateLimited`, `ErrNotWhitelisted`,
  `ErrDuplicateTransaction`, `ErrInconsistentTransaction`, `ErrInvalidSignature`,
  `ErrInvalidCallback`.
- `APIStatus.TranID`, documented as a gateway reference for check and details.
- `hash.Generator.Equal` for constant-time signature checks.
- Payment status constants (`StatusApproved`, `StatusPreAuth`, ...).

### Changed

- Redirects are no longer followed unless your `HTTPClient` sets
  `CheckRedirect`; following one would resend the signed request elsewhere.
- The default HTTP timeout is 10s (was 30s). `Config.HTTPTimeout` overrides it.
- `CheckTransactionResponse.IsApproved` now also requires status code 0, so a
  `PRE-AUTH` or conflicting response is not reported as paid.
- `GetDetails(ctx, nil)` returns an error instead of panicking.

### Deprecated

- `CallbackPayload`, `Callback.ParseAndVerify` and `Callback.Verify`: they
  expect the hash in the body, a string status and a fixed field list, which
  does not match PayWay's signed callbacks. Use `VerifyRequest`/`VerifyBody`.
- Float amount fields in `CheckTransactionData` and `TransactionDetailsData`:
  use the exact `.Transaction` view.
