package template

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(body))
}

// buildServer answers the whole build sequence. statuses is served one per
// status poll, so a test can walk a build from building to ready.
type buildServer struct {
	statuses []map[string]any

	mux         *http.ServeMux
	statusCalls atomic.Int32
	createBody  map[string]any
	startBody   map[string]any
	patchBody   map[string]any
}

func newBuildServer(t *testing.T, statuses []map[string]any) *buildServer {
	bs := &buildServer{statuses: statuses, mux: http.NewServeMux()}

	bs.mux.HandleFunc("POST /v3/templates", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&bs.createBody))
		writeJSON(t, w, http.StatusAccepted, map[string]any{
			"templateID": "tpl-1",
			"buildID":    "bld-1",
			"public":     false,
			"names":      []string{"my-python"},
			"aliases":    []string{},
			"tags":       []string{"v1"},
		})
	})

	bs.mux.HandleFunc("POST /v2/templates/tpl-1/builds/bld-1", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&bs.startBody))
		w.WriteHeader(http.StatusAccepted)
	})

	// A nil statuses leaves the status route free for the test to register
	// its own.
	if statuses != nil {
		bs.mux.HandleFunc("GET /templates/tpl-1/builds/bld-1/status", func(w http.ResponseWriter, _ *http.Request) {
			i := int(bs.statusCalls.Add(1)) - 1
			if i >= len(bs.statuses) {
				i = len(bs.statuses) - 1
			}
			writeJSON(t, w, http.StatusOK, bs.statuses[i])
		})
	}

	bs.mux.HandleFunc("PATCH /v2/templates/tpl-1", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&bs.patchBody))
		writeJSON(t, w, http.StatusOK, map[string]any{"names": []string{"my-python"}})
	})

	return bs
}

// status builds one status-poll response.
func status(state string, entries ...map[string]any) map[string]any {
	if entries == nil {
		entries = []map[string]any{}
	}
	return map[string]any{
		"templateID": "tpl-1",
		"buildID":    "bld-1",
		"status":     state,
		"logs":       []string{},
		"logEntries": entries,
	}
}

func logEntry(level, message string) map[string]any {
	return map[string]any{
		"timestamp": "2026-09-01T10:00:00Z",
		"level":     level,
		"message":   message,
	}
}

func TestBuildHappyPath(t *testing.T) {
	bs := newBuildServer(t, []map[string]any{
		status("building", logEntry("info", "step 1")),
		status("building", logEntry("info", "step 2")),
		status("ready"),
	})
	svc := newTestService(t, bs.mux)

	var logs []string
	info, err := svc.Build(context.Background(),
		New(BuilderOptions{Region: "cn-wlcb"}).RunCmd("echo hi"),
		"my-python",
		BuildOptions{
			CPUCount: 2,
			MemoryMB: 2048,
			Tags:     []string{"v1"},
			OnLogs:   func(e LogEntry) { logs = append(logs, e.Message) },
		})
	require.NoError(t, err)

	assert.Equal(t, "tpl-1", info.TemplateID)
	assert.Equal(t, "bld-1", info.BuildID)
	assert.Equal(t, "my-python", info.Name)
	assert.Equal(t, []string{"v1"}, info.Tags)

	assert.Equal(t, map[string]any{
		"name":     "my-python",
		"tags":     []any{"v1"},
		"cpuCount": float64(2),
		"memoryMB": float64(2048),
	}, bs.createBody)

	assert.Equal(t, CNBaseImage, bs.startBody["fromImage"])

	// The build's own log lines reach the callback, alongside the SDK's
	// progress messages.
	assert.Contains(t, logs, "step 1")
	assert.Contains(t, logs, "step 2")
	assert.Contains(t, logs, "Template created with ID tpl-1, build bld-1")
}

func TestBuildReportsFailureWithTheBuildID(t *testing.T) {
	bs := newBuildServer(t, []map[string]any{
		{
			"templateID": "tpl-1",
			"buildID":    "bld-1",
			"status":     "error",
			"logs":       []string{},
			"logEntries": []map[string]any{},
			"reason": map[string]any{
				"message": "step 3 failed: exit code 1",
				"step":    "RUN pip install",
			},
		},
	})
	svc := newTestService(t, bs.mux)

	info, err := svc.Build(context.Background(), New(BuilderOptions{}), "my-python", BuildOptions{})
	require.Error(t, err)

	// A failed build still has an ID worth reporting, so info comes back too.
	require.NotNil(t, info)
	assert.Equal(t, "bld-1", info.BuildID)

	var buildErr *errdefs.BuildError
	require.ErrorAs(t, err, &buildErr)
	assert.Equal(t, "bld-1", buildErr.BuildID)
	assert.Equal(t, "tpl-1", buildErr.TemplateID)
	assert.Contains(t, buildErr.Error(), "step 3 failed")
}

func TestBuildPublishes(t *testing.T) {
	bs := newBuildServer(t, []map[string]any{status("ready")})
	svc := newTestService(t, bs.mux)

	_, err := svc.Build(context.Background(), New(BuilderOptions{}), "my-python", BuildOptions{Publish: true})
	require.NoError(t, err)

	assert.Equal(t, map[string]any{"public": true}, bs.patchBody)
}

func TestBuildDoesNotPublishByDefault(t *testing.T) {
	bs := newBuildServer(t, []map[string]any{status("ready")})
	svc := newTestService(t, bs.mux)

	_, err := svc.Build(context.Background(), New(BuilderOptions{}), "my-python", BuildOptions{})
	require.NoError(t, err)

	assert.Nil(t, bs.patchBody, "a template must not become public unless asked")
}

func TestBuildSkipCache(t *testing.T) {
	bs := newBuildServer(t, []map[string]any{status("ready")})
	svc := newTestService(t, bs.mux)

	_, err := svc.Build(context.Background(), New(BuilderOptions{}).RunCmd("x"), "t", BuildOptions{SkipCache: true})
	require.NoError(t, err)

	assert.Equal(t, true, bs.startBody["force"])
}

func TestBuildUploadsCopyBundlesOnce(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.py"), []byte("print('hi')"), 0o644))

	bs := newBuildServer(t, []map[string]any{status("ready")})

	var uploads atomic.Int32
	var linkRequests []string
	uploadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		uploads.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer uploadSrv.Close()

	bs.mux.HandleFunc("GET /templates/tpl-1/files/{hash}", func(w http.ResponseWriter, r *http.Request) {
		linkRequests = append(linkRequests, r.PathValue("hash"))
		url := uploadSrv.URL + "/put"
		writeJSON(t, w, http.StatusCreated, map[string]any{"present": false, "url": url})
	})

	svc := newTestService(t, bs.mux)

	// Two COPY steps sharing a source but not a destination. The destination
	// is part of the hash, so these are two distinct bundles.
	b := New(BuilderOptions{FileContextPath: dir}).
		Copy("app.py", "/app/app.py").
		Copy("app.py", "/backup/app.py")

	_, err := svc.Build(context.Background(), b, "t", BuildOptions{})
	require.NoError(t, err)

	require.Len(t, linkRequests, 2, "each distinct COPY destination has its own hash")
	assert.NotEqual(t, linkRequests[0], linkRequests[1],
		"the destination is part of the hash, so the two steps differ")
	assert.EqualValues(t, 2, uploads.Load())

	steps := bs.startBody["steps"].([]any)
	require.Len(t, steps, 2)
	assert.NotEmpty(t, steps[0].(map[string]any)["filesHash"])
}

func TestBuildSkipsUploadWhenBundleIsPresent(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.py"), []byte("print('hi')"), 0o644))

	bs := newBuildServer(t, []map[string]any{status("ready")})
	bs.mux.HandleFunc("GET /templates/tpl-1/files/{hash}", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusCreated, map[string]any{"present": true})
	})

	svc := newTestService(t, bs.mux)

	b := New(BuilderOptions{FileContextPath: dir}).Copy("app.py", "/app/app.py")
	_, err := svc.Build(context.Background(), b, "t", BuildOptions{})
	require.NoError(t, err)
}

func TestBuildRejectsCopyWithoutFileContext(t *testing.T) {
	bs := newBuildServer(t, []map[string]any{status("ready")})
	svc := newTestService(t, bs.mux)

	_, err := svc.Build(context.Background(),
		New(BuilderOptions{}).Copy("app.py", "/app.py"), "t", BuildOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FileContextPath")

	// The failure has to happen before anything is registered server-side.
	assert.Nil(t, bs.createBody)
}

func TestCopySourceCannotEscapeTheFileContext(t *testing.T) {
	dir := t.TempDir()

	for _, src := range []string{"../outside.txt", "/etc/passwd"} {
		_, err := New(BuilderOptions{FileContextPath: dir}).Copy(src, "/x").prepareSteps()
		require.Error(t, err, "source %q", src)
	}
}

func TestWaitForBuildAdvancesTheLogOffset(t *testing.T) {
	var offsets []string
	bs := newBuildServer(t, nil)
	bs.mux.HandleFunc("GET /templates/tpl-1/builds/bld-1/status", func(w http.ResponseWriter, r *http.Request) {
		offsets = append(offsets, r.URL.Query().Get("logsOffset"))
		switch len(offsets) {
		case 1:
			writeJSON(t, w, http.StatusOK, status("building", logEntry("info", "a"), logEntry("info", "b")))
		case 2:
			writeJSON(t, w, http.StatusOK, status("building", logEntry("info", "c")))
		default:
			writeJSON(t, w, http.StatusOK, status("ready"))
		}
	})

	svc := newTestService(t, bs.mux)

	var seen []string
	_, err := svc.WaitForBuild(context.Background(), "tpl-1", "bld-1", func(e LogEntry) {
		seen = append(seen, e.Message)
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"a", "b", "c"}, seen, "each entry should be delivered once")
	// The first poll sends no offset; later ones skip what has been seen.
	assert.Equal(t, []string{"", "2", "3"}, offsets)
}

func TestWaitForBuildStopsOnContextCancel(t *testing.T) {
	bs := newBuildServer(t, []map[string]any{status("building")})
	svc := newTestService(t, bs.mux)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.WaitForBuild(ctx, "tpl-1", "bld-1", nil)
	assert.ErrorIs(t, err, context.Canceled)
}
