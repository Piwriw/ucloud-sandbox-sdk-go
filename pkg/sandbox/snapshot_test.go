package sandbox

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSnapshot(t *testing.T) {
	var gotPath string
	var gotBody map[string]any

	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		decodeBody(t, r, &gotBody)
		writeJSON(t, w, http.StatusCreated, map[string]any{
			"snapshotID": "team/nightly:default",
			"names":      []string{"team/nightly:default"},
		})
	}))

	info, err := svc.CreateSnapshot(context.Background(), "sbx-1", SnapshotOptions{Name: "nightly"})
	require.NoError(t, err)

	assert.Equal(t, "/sandboxes/sbx-1/snapshots", gotPath)
	assert.Equal(t, map[string]any{"name": "nightly"}, gotBody)
	assert.Equal(t, "team/nightly:default", info.SnapshotID)
	assert.Equal(t, []string{"team/nightly:default"}, info.Names)
}

func TestDeleteSnapshot(t *testing.T) {
	t.Run("existing snapshot", func(t *testing.T) {
		var gotPath, gotMethod string
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath, gotMethod = r.URL.Path, r.Method
			w.WriteHeader(http.StatusNoContent)
		}))

		ok, err := svc.DeleteSnapshot(context.Background(), "snap-1")
		require.NoError(t, err)
		assert.True(t, ok)

		// There is no snapshot-delete endpoint: a snapshot is a template, and
		// this is how the platform's own SDKs remove one.
		assert.Equal(t, http.MethodDelete, gotMethod)
		assert.Equal(t, "/templates/snap-1", gotPath)
	})

	t.Run("a namespaced snapshot ID is escaped, not split into path segments", func(t *testing.T) {
		var gotEscaped, gotDecoded string
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotEscaped, gotDecoded = r.URL.EscapedPath(), r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		}))

		// A snapshot ID carries a namespace and a tag, so it contains a slash.
		// Left unescaped it would read as two path segments and address the
		// wrong resource.
		_, err := svc.DeleteSnapshot(context.Background(), "team-slug/my-snapshot:default")
		require.NoError(t, err)

		assert.Equal(t, "/templates/team-slug%2Fmy-snapshot:default", gotEscaped)
		assert.Equal(t, "/templates/team-slug/my-snapshot:default", gotDecoded)
	})

	t.Run("missing snapshot is false, not an error", func(t *testing.T) {
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))

		ok, err := svc.DeleteSnapshot(context.Background(), "gone")
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestSnapshotExists(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		var gotQuery string
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			writeJSON(t, w, http.StatusOK, []map[string]any{{"snapshotID": "snap-1", "names": []string{}}})
		}))

		ok, err := svc.SnapshotExists(context.Background(), "snap-1")
		require.NoError(t, err)
		assert.True(t, ok)

		// The server filters, so this is one request however many snapshots
		// the team has. Earlier versions walked every page.
		assert.Contains(t, gotQuery, "name=snap-1")
		assert.Contains(t, gotQuery, "limit=1")
	})

	t.Run("missing", func(t *testing.T) {
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, []map[string]any{})
		}))

		ok, err := svc.SnapshotExists(context.Background(), "gone")
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("a server failure is not reported as absence", func(t *testing.T) {
		svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))

		_, err := svc.SnapshotExists(context.Background(), "snap-1")
		assert.Error(t, err)
	})
}

func TestSandboxListSnapshotsFiltersToItself(t *testing.T) {
	var gotQuery string
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, http.StatusOK, []map[string]any{{"snapshotID": "snap-1", "names": []string{}}})
	}))

	sbx, err := svc.newSandbox("sbx-7", apiSandbox("0.4.0"))
	require.NoError(t, err)

	snapshots, err := sbx.ListSnapshots(context.Background(), ListSnapshotsOptions{})
	require.NoError(t, err)

	assert.Contains(t, gotQuery, "sandboxID=sbx-7")
	require.Len(t, snapshots, 1)
	assert.Equal(t, "snap-1", snapshots[0].SnapshotID)
}
