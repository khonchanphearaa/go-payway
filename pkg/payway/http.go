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

const defaultTimeout = 30 * time.Second

// httpClient is an internal HTTP wrapper used by all service types.
type httpClient struct {
	base   string
	client *http.Client
}

func newHTTPClient(cfg Config) *httpClient {
	timeout := cfg.HTTPTimeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	return &httpClient{
		base:   cfg.baseURL(),
		client: &http.Client{Timeout: timeout},
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

// do executes the request and handles common errors.
func (h *httpClient) do(req *http.Request, dest any) error {
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("payway: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("payway: failed to read response: %w", err)
	}

	// PayWay occasionally returns HTML on hard errors (e.g. 405, wrong domain)
	if resp.Header.Get("Content-Type") != "" &&
		strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return fmt.Errorf("payway: unexpected HTML response (status %d) — check your domain whitelist or request method", resp.StatusCode)
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("payway: failed to decode response: %w (body: %s)", err, truncate(string(raw), 200))
	}

	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
