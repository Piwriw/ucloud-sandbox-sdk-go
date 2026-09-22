package transport

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"
)

// maxRetryAfter caps how long a single Retry-After is honoured. A server asking
// for longer is treated as asking for more than a client call should wait, and
// the 429 is returned instead.
const maxRetryAfter = 60 * time.Second

// retryTransport retries a 429 that asks to be retried, and nothing else.
//
// Only 429 is retried, and only when Retry-After gives a plain number of
// seconds. The control plane's other failures are not known to be idempotent,
// and an HTTP-date Retry-After is not worth the clock-skew risk — both are
// passed straight through. This mirrors the Python SDK's retry rule.
type retryTransport struct {
	base    http.RoundTripper
	retries int
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		resp, err := t.base.RoundTrip(req)
		if err != nil || resp.StatusCode != http.StatusTooManyRequests {
			return resp, err
		}
		if attempt >= t.retries {
			return resp, nil
		}

		wait, ok := retryAfter(resp)
		if !ok {
			return resp, nil
		}

		// A retry re-sends the body, so it needs one that can be replayed.
		// http.NewRequest sets GetBody for the in-memory body types the SDK
		// uses; anything else (a streaming upload, say) is not retried.
		body, err := replayBody(req)
		if err != nil {
			return resp, nil
		}

		drain(resp)

		if err := sleep(req.Context(), wait); err != nil {
			return nil, err
		}

		req = req.Clone(req.Context())
		req.Body = body
	}
}

// retryAfter reads a Retry-After holding a non-negative number of seconds,
// within maxRetryAfter.
func retryAfter(resp *http.Response) (time.Duration, bool) {
	raw := resp.Header.Get("Retry-After")
	if raw == "" {
		return 0, false
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 {
		return 0, false
	}
	wait := time.Duration(seconds) * time.Second
	if wait > maxRetryAfter {
		return 0, false
	}
	return wait, true
}

// replayBody produces a fresh reader for the request body, or reports that the
// body cannot be replayed. A request with no body replays trivially.
func replayBody(req *http.Request) (io.ReadCloser, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return req.Body, nil
	}
	if req.GetBody == nil {
		return nil, errNoReplayableBody
	}
	return req.GetBody()
}

// errNoReplayableBody reports a request whose body cannot be sent twice, so the
// 429 is surfaced to the caller instead of being retried.
var errNoReplayableBody = errors.New("transport: request body cannot be replayed")

// drain reads and closes a response body so the connection can be reused.
func drain(resp *http.Response) {
	if resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	_ = resp.Body.Close()
}

// sleep waits for d, or returns early if the context ends first.
func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
