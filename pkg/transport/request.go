package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// Do sends a request the control-plane spec does not describe, applying the
// same authentication and headers the generated client uses.
//
// The volume content API is the only surface that needs this: it lives behind a
// separate base URL and is absent from both OpenAPI specs. Everything the spec
// covers should go through API() instead, so that request shapes stay tied to
// the spec.
//
// The caller owns the returned response body. A non-2xx status is returned as
// an error from errdefs, with the body as its message, and its body is closed.
func (c *Client) Do(ctx context.Context, method, url string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if err := c.authorize(ctx, req); err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return nil, errdefs.FromHTTP(resp.StatusCode, string(respBody))
	}
	return resp, nil
}

// DoJSON is Do with JSON on both ends. body is marshalled when non-nil, and the
// response is decoded into result when non-nil and the response carries one.
func (c *Client) DoJSON(ctx context.Context, method, url string, body, result any) error {
	var reader io.Reader
	contentType := ""
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
		contentType = "application/json"
	}

	resp, err := c.Do(ctx, method, url, reader, contentType)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(result)
}
