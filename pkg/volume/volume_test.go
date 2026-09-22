package volume

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

func newTestService(t *testing.T, handler http.Handler) *Service {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	tc, err := transport.New(transport.Config{APIKey: "test-key", APIURL: srv.URL})
	require.NoError(t, err)

	return NewService(tc)
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
	var gotPath, gotMethod string
	var gotBody map[string]any

	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		writeJSON(t, w, http.StatusCreated, map[string]any{
			"volumeID": "vol-1",
			"name":     "datasets",
			"token":    "volume-token",
		})
	}))

	v, err := svc.Create(context.Background(), "datasets")
	require.NoError(t, err)

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/volumes", gotPath)
	assert.Equal(t, map[string]any{"name": "datasets"}, gotBody)

	assert.Equal(t, "vol-1", v.ID)
	assert.Equal(t, "datasets", v.Name)
}

func TestGetReturnsTheContentToken(t *testing.T) {
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/volumes/vol-1", r.URL.Path)
		writeJSON(t, w, http.StatusOK, map[string]any{
			"volumeID": "vol-1",
			"name":     "datasets",
			"token":    "volume-token",
		})
	}))

	// The token is not on the Volume handle, since this SDK wraps no operation
	// that uses it. Get still surfaces it for callers that reach the upstream
	// content API themselves.
	got, err := svc.Get(context.Background(), "vol-1")
	require.NoError(t, err)
	assert.Equal(t, "volume-token", got.Token)
}

func TestConnect(t *testing.T) {
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]any{
			"volumeID": "vol-1",
			"name":     "datasets",
			"token":    "volume-token",
		})
	}))

	v, err := svc.Connect(context.Background(), "vol-1")
	require.NoError(t, err)
	assert.Equal(t, "vol-1", v.ID)
	assert.Equal(t, "datasets", v.Name)
}

func TestList(t *testing.T) {
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/volumes", r.URL.Path)
		writeJSON(t, w, http.StatusOK, []map[string]any{
			{"volumeID": "vol-1", "name": "datasets"},
			{"volumeID": "vol-2", "name": "models"},
		})
	}))

	volumes, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []Info{
		{VolumeID: "vol-1", Name: "datasets"},
		{VolumeID: "vol-2", Name: "models"},
	}, volumes)
}

func TestListEmptyIsNeverNil(t *testing.T) {
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusOK, []map[string]any{})
	}))

	volumes, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, volumes)
	assert.Empty(t, volumes)
}

func TestDelete(t *testing.T) {
	t.Run("existing volume", func(t *testing.T) {
		var gotPath, gotMethod string
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath, gotMethod = r.URL.Path, r.Method
			w.WriteHeader(http.StatusNoContent)
		}))

		ok, err := svc.Delete(context.Background(), "vol-1")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, http.MethodDelete, gotMethod)
		assert.Equal(t, "/volumes/vol-1", gotPath)
	})

	t.Run("missing volume is false, not an error", func(t *testing.T) {
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))

		ok, err := svc.Delete(context.Background(), "gone")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("other failures are errors", func(t *testing.T) {
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))

		_, err := svc.Delete(context.Background(), "vol-1")
		require.Error(t, err)
		assert.ErrorIs(t, err, errdefs.ErrAuth)
	})
}

func TestDestroyUsesTheHandlesID(t *testing.T) {
	var gotPath string
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))

	v := &Volume{ID: "vol-7", Name: "scratch", svc: svc}
	ok, err := v.Destroy(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "/volumes/vol-7", gotPath)
}

func TestMount(t *testing.T) {
	v := &Volume{ID: "vol-1", Name: "datasets"}

	// The mount travels by name, not by ID, and uses the generated type so
	// pkg/sandbox need not import this package.
	assert.Equal(t,
		api.SandboxVolumeMount{Name: "datasets", Path: "/mnt/data"},
		v.Mount("/mnt/data"))
}
