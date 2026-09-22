package transport

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// retryClient builds a client whose retry budget is n.
func retryClient(n int) *http.Client {
	return &http.Client{Transport: &retryTransport{base: http.DefaultTransport, retries: n}}
}

func TestRetryOn429WithRetryAfter(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	resp, err := retryClient(3).Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.EqualValues(t, 3, calls.Load(), "server should have seen two retries")
}

func TestRetryStopsAtBudget(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	resp, err := retryClient(2).Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode,
		"the last 429 should reach the caller")
	assert.EqualValues(t, 3, calls.Load(), "one initial attempt plus two retries")
}

func TestNoRetryWithoutUsableRetryAfter(t *testing.T) {
	tests := map[string]string{
		"absent":    "",
		"http date": "Wed, 21 Oct 2026 07:28:00 GMT",
		"negative":  "-5",
		"too long":  "3600",
		"garbage":   "soon",
	}

	for name, header := range tests {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				if header != "" {
					w.Header().Set("Retry-After", header)
				}
				w.WriteHeader(http.StatusTooManyRequests)
			}))
			defer srv.Close()

			resp, err := retryClient(3).Get(srv.URL)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.EqualValues(t, 1, calls.Load(), "should not retry")
		})
	}
}

func TestNoRetryOnOtherStatuses(t *testing.T) {
	statuses := []int{
		http.StatusOK,
		http.StatusBadRequest,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	}

	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(status)
			}))
			defer srv.Close()

			resp, err := retryClient(3).Get(srv.URL)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.EqualValues(t, 1, calls.Load(), "only 429 is retried")
		})
	}
}

func TestRetryReplaysRequestBody(t *testing.T) {
	var calls atomic.Int32
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		if calls.Add(1) < 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// http.NewRequest sets GetBody for a *strings.Reader, which is what makes
	// the replay possible.
	req, err := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(`{"name":"x"}`))
	require.NoError(t, err)

	resp, err := retryClient(3).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, []string{`{"name":"x"}`, `{"name":"x"}`}, bodies,
		"the retry should send the same body as the first attempt")
}

func TestNoRetryWhenBodyCannotBeReplayed(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	// A pipe has no GetBody, so the body is gone once it has been sent.
	pr, pw := io.Pipe()
	go func() {
		_, _ = pw.Write([]byte("streamed"))
		_ = pw.Close()
	}()

	req, err := http.NewRequest(http.MethodPost, srv.URL, pr)
	require.NoError(t, err)

	resp, err := retryClient(3).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.EqualValues(t, 1, calls.Load(),
		"a body that cannot be replayed must not be retried")
}
