package secret

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// newTestService points a Service at handler and returns it.
func newTestService(t *testing.T, handler http.Handler) *Service {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	tc, err := transport.New(transport.Config{APIKey: "test-key", APIURL: srv.URL})
	require.NoError(t, err)

	return NewService(tc)
}

// secretJSON is the wire shape of a Secret, spelled out rather than built from
// the generated type so that a change to either side shows up as a test
// failure.
func secretJSON(id, name string, version int, metadata map[string]string) map[string]any {
	if metadata == nil {
		metadata = map[string]string{}
	}
	return map[string]any{
		"secretID":       id,
		"name":           name,
		"currentVersion": version,
		"metadata":       metadata,
		"createdAt":      "2026-09-01T10:00:00Z",
		"updatedAt":      "2026-09-02T11:30:00Z",
	}
}

// writeJSON answers with a JSON body. The generated client dispatches on the
// response's content type, so a handler that omits it parses as nothing.
func writeJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(body))
}

func TestCreate(t *testing.T) {
	var gotPath, gotMethod, gotAPIKey string
	var gotBody map[string]any

	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		gotAPIKey = r.Header.Get("X-API-Key")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))

		writeJSON(t, w, http.StatusCreated, secretJSON("sec_1", "openai-key", 1, map[string]string{"env": "prod"}))
	}))

	info, err := svc.Create(context.Background(), "openai-key", "sk-secret", CreateOptions{
		Metadata: map[string]string{"env": "prod"},
	})
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/secrets", gotPath)
	assert.Equal(t, "test-key", gotAPIKey)
	assert.Equal(t, map[string]any{
		"name":     "openai-key",
		"value":    "sk-secret",
		"metadata": map[string]any{"env": "prod"},
	}, gotBody)

	assert.Equal(t, "sec_1", info.SecretID)
	assert.Equal(t, "openai-key", info.Name)
	assert.EqualValues(t, 1, info.Version)
	assert.Equal(t, map[string]string{"env": "prod"}, info.Metadata)
	assert.Equal(t, 2026, info.CreatedAt.Year())
}

func TestCreateOmitsUnsetMetadata(t *testing.T) {
	var gotBody map[string]any

	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		writeJSON(t, w, http.StatusCreated, secretJSON("sec_1", "k", 1, nil))
	}))

	_, err := svc.Create(context.Background(), "k", "v")
	require.NoError(t, err)

	// Leaving metadata unset must not send an empty map, which would clear
	// whatever the server would otherwise default to.
	assert.NotContains(t, gotBody, "metadata")
}

func TestCreateRejectsBadNameBeforeSending(t *testing.T) {
	var called bool
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))

	for _, name := range []string{"", "has space", "has/slash", "has{brace}"} {
		_, err := svc.Create(context.Background(), name, "v")
		assert.ErrorIs(t, err, errdefs.ErrInvalidArgument, "name %q", name)
	}
	assert.False(t, called, "an invalid name should not reach the server")
}

func TestGetInfo(t *testing.T) {
	var gotPath string
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(t, w, http.StatusOK, secretJSON("sec_1", "openai-key", 3, nil))
	}))

	info, err := svc.GetInfo(context.Background(), "openai-key")
	require.NoError(t, err)

	assert.Equal(t, "/secrets/openai-key", gotPath, "a name is accepted in place of an ID")
	assert.EqualValues(t, 3, info.Version)
	assert.NotNil(t, info.Metadata, "metadata should be non-nil even when empty")
	assert.Empty(t, info.Metadata)
}

func TestGetInfoNotFound(t *testing.T) {
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":404,"message":"secret not found"}`))
	}))

	_, err := svc.GetInfo(context.Background(), "missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, errdefs.ErrNotFound)
	assert.Contains(t, err.Error(), "secret not found")
}

func TestUpdate(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any

	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		writeJSON(t, w, http.StatusOK, secretJSON("sec_1", "openai-key", 2, nil))
	}))

	info, err := svc.Update(context.Background(), "sec_1", "sk-rotated")
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, gotMethod, "update is a POST, not a PUT or PATCH")
	assert.Equal(t, "/secrets/sec_1", gotPath)
	assert.Equal(t, map[string]any{"value": "sk-rotated"}, gotBody)
	assert.EqualValues(t, 2, info.Version)
}

func TestUpdateClearsMetadataWithEmptyMap(t *testing.T) {
	var gotBody map[string]any

	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		writeJSON(t, w, http.StatusOK, secretJSON("sec_1", "k", 2, nil))
	}))

	_, err := svc.Update(context.Background(), "sec_1", "v", UpdateOptions{
		Metadata: map[string]string{},
	})
	require.NoError(t, err)

	// An empty non-nil map is how a caller asks to clear the metadata, so it
	// has to reach the wire.
	assert.Equal(t, map[string]any{}, gotBody["metadata"])
}

func TestDelete(t *testing.T) {
	t.Run("existing secret", func(t *testing.T) {
		var gotPath, gotMethod string
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath, gotMethod = r.URL.Path, r.Method
			w.WriteHeader(http.StatusNoContent)
		}))

		ok, err := svc.Delete(context.Background(), "sec_1")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, http.MethodDelete, gotMethod)
		assert.Equal(t, "/secrets/sec_1", gotPath)
	})

	t.Run("missing secret is false, not an error", func(t *testing.T) {
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))

		ok, err := svc.Delete(context.Background(), "gone")
		require.NoError(t, err, "deleting twice should not be a failure")
		assert.False(t, ok)
	})

	t.Run("other failures are errors", func(t *testing.T) {
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))

		ok, err := svc.Delete(context.Background(), "sec_1")
		require.Error(t, err)
		assert.ErrorIs(t, err, errdefs.ErrForbidden)
		assert.False(t, ok)
	})
}

func TestExists(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, secretJSON("sec_1", "k", 1, nil))
		}))
		ok, err := svc.Exists(context.Background(), "k")
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("missing", func(t *testing.T) {
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		ok, err := svc.Exists(context.Background(), "k")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("a server failure is not reported as absence", func(t *testing.T) {
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		_, err := svc.Exists(context.Background(), "k")
		assert.Error(t, err)
	})
}

func TestListPaginates(t *testing.T) {
	var gotQueries []string

	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQueries = append(gotQueries, r.URL.RawQuery)

		switch r.URL.Query().Get("nextToken") {
		case "":
			w.Header().Set(transport.NextTokenHeader, "cursor-2")
			writeJSON(t, w, http.StatusOK, []map[string]any{
				secretJSON("sec_1", "a", 1, nil),
				secretJSON("sec_2", "b", 1, nil),
			})
		case "cursor-2":
			// No X-Next-Token: this is the last page.
			writeJSON(t, w, http.StatusOK, []map[string]any{
				secretJSON("sec_3", "c", 1, nil),
			})
		default:
			t.Errorf("unexpected cursor %q", r.URL.Query().Get("nextToken"))
		}
	}))

	all, err := svc.List(context.Background(), ListOptions{Limit: 2}).All(context.Background())
	require.NoError(t, err)

	names := make([]string, len(all))
	for i, s := range all {
		names[i] = s.Name
	}
	assert.Equal(t, []string{"a", "b", "c"}, names,
		"the paginator must follow X-Next-Token rather than stopping after one page")

	require.Len(t, gotQueries, 2)
	assert.Equal(t, "limit=2", gotQueries[0])
	assert.Contains(t, gotQueries[1], "nextToken=cursor-2")
	assert.Contains(t, gotQueries[1], "limit=2")
}

func TestListStopsWithoutCursor(t *testing.T) {
	var calls int
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		writeJSON(t, w, http.StatusOK, []map[string]any{secretJSON("sec_1", "a", 1, nil)})
	}))

	all, err := svc.List(context.Background()).All(context.Background())
	require.NoError(t, err)
	assert.Len(t, all, 1)
	assert.Equal(t, 1, calls, "a page without a cursor ends the listing")
}

func TestFill(t *testing.T) {
	ref, err := Fill("openai-key")
	require.NoError(t, err)
	assert.Equal(t, "${e2b.secrets.openai-key}", ref)

	assert.Equal(t, "${e2b.secrets.openai-key}", MustFill("openai-key"))
}

func TestFillRejectsNamesThatWouldBreakThePlaceholder(t *testing.T) {
	// "a}b" would render as "${e2b.secrets.a}b}", which silently resolves to
	// the secret named "a" -- the wrong secret, with no error anywhere.
	for _, name := range []string{"a}b", "a{b", "", "a b", "a.b"} {
		_, err := Fill(name)
		assert.ErrorIs(t, err, errdefs.ErrInvalidArgument, "name %q", name)
	}

	assert.Panics(t, func() { MustFill("a}b") })
}

func TestValidateName(t *testing.T) {
	valid := []string{"a", "openai-key", "OPENAI_KEY", "k8s-token-2", fmt.Sprintf("%0*d", MaxNameLength, 0)}
	for _, name := range valid {
		assert.NoError(t, ValidateName(name), "name %q", name)
	}

	invalid := []string{"", "with space", "with/slash", "with.dot", fmt.Sprintf("%0*d", MaxNameLength+1, 0)}
	for _, name := range invalid {
		assert.ErrorIs(t, ValidateName(name), errdefs.ErrInvalidArgument, "name %q", name)
	}
}
