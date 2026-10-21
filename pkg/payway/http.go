package payway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 10 * time.Second

// maxResponseBytes bounds how much of a PayWay response the SDK will read.
const maxResponseBytes = 2 << 20

// httpClient is an internal HTTP wrapper used by all service types.
type httpClient struct {
	base   string
	client *http.Client
}

func newHTTPClient(cfg Config) *httpClient {
	c := &http.Client{}
	if cfg.HTTPClient != nil {
		copied := *cfg.HTTPClient
		c = &copied
	}
	if cfg.HTTPTimeout > 0 {
		c.Timeout = cfg.HTTPTimeout
	} else if c.Timeout == 0 {
		c.Timeout = defaultTimeout
	}
	// Following a 307/308 would resend the signed request body to another
	// host. Refuse redirects unless the caller configured its own policy.
	if c.CheckRedirect == nil {
		c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	return &httpClient{
		base:   cfg.baseURL(),
		client: c,
	}
}

// postJSON sends a JSON POST request and decodes the response into dest.
func (h *httpClient) postJSON(ctx context.Context, path string, body any, dest any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("payway: failed to encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.base+path, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("payway: failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return h.do(req, dest)
}

// postForm sends a multipart/form-data POST request and decodes the JSON response.
func (h *httpClient) postForm(ctx context.Context, path string, fields map[string]string, dest any) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for key, val := range fields {
		if val == "" {
			continue
		}
		if err := w.WriteField(key, val); err != nil {
			return fmt.Errorf("payway: failed to write field %q: %w", key, err)
		}
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.base+path, &buf)
	if err != nil {
		return fmt.Errorf("payway: failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	return h.do(req, dest)
}

// do executes the request and handles common errors. Errors never include the
// raw response body; only PayWay's status code, message and trace ID.
func (h *httpClient) do(req *http.Request, dest any) error {
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("payway: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("payway: failed to read response: %w", err)
	}
	if len(raw) > maxResponseBytes {
		return fmt.Errorf("payway: response exceeds %d bytes", maxResponseBytes)
	}

	// PayWay occasionally returns HTML on hard errors (e.g. 405, wrong domain)
	if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return fmt.Errorf("payway: unexpected HTML response (status %d) — check your domain whitelist or request method", resp.StatusCode)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Error bodies usually carry the status envelope with PayWay's reason.
		var envelope struct {
			Status APIStatus `json:"status"`
		}
		if json.Unmarshal(raw, &envelope) == nil && (envelope.Status.Code != "" || envelope.Status.Message != "") {
			e := statusError(req.URL.Path, envelope.Status)
			e.HTTPStatus = resp.StatusCode
			return e
		}
		return &Error{HTTPStatus: resp.StatusCode, Message: http.StatusText(resp.StatusCode), path: req.URL.Path}
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("payway: failed to decode response (status %d): %w", resp.StatusCode, err)
	}

	return nil
}
