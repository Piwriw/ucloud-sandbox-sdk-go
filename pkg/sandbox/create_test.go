package sandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

func newTestService(t *testing.T, handler http.Handler) *Service {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	tc, err := transport.New(transport.Config{
		APIKey:     "test-key",
		APIURL:     srv.URL,
		SandboxURL: srv.URL,
	})
	require.NoError(t, err)

	return NewService(tc)
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(body))
}

// createdSandbox is the control plane's answer to a create.
func createdSandbox() map[string]any {
	return map[string]any{
		"sandboxID":   "sbx-1",
		"templateID":  "base",
		"clientID":    "client-1",
		"envdVersion": "0.4.0",
	}
}

// captureCreate returns a service whose create requests land in the returned
// map.
func captureCreate(t *testing.T, body *map[string]any) *Service {
	t.Helper()
	return newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(body))
		writeJSON(t, w, http.StatusCreated, createdSandbox())
	}))
}

func TestCreateDefaults(t *testing.T) {
	var got map[string]any
	svc := captureCreate(t, &got)

	sbx, err := svc.Create(context.Background(), CreateOptions{})
	require.NoError(t, err)
	assert.Equal(t, "sbx-1", sbx.ID)

	assert.Equal(t, "base", got["templateID"])
	assert.EqualValues(t, DefaultTimeoutSeconds, got["timeout"])

	// Every sandbox the SDK opens is marked, so the platform can tell which
	// product opened it.
	metadata := got["metadata"].(map[string]any)
	assert.Equal(t, ManageByDefault, metadata[ManageByMetadataKey])
}

func TestCreateInternetAccessUsesTheSpecsFieldName(t *testing.T) {
	var got map[string]any
	svc := captureCreate(t, &got)

	_, err := svc.Create(context.Background(), CreateOptions{AllowInternetAccess: new(false)})
	require.NoError(t, err)

	// The field is snake_case in the spec while its neighbours are camelCase.
	// The previous SDK sent allowInternetAccess, which the platform ignored --
	// so a sandbox asked to have no internet access got it anyway.
	assert.Equal(t, false, got["allow_internet_access"])
	assert.NotContains(t, got, "allowInternetAccess")
}

func TestCreatePassesThroughNewFields(t *testing.T) {
	var got map[string]any
	svc := captureCreate(t, &got)

	_, err := svc.Create(context.Background(), CreateOptions{
		Template:        "python",
		TimeoutSeconds:  600,
		AutoPause:       new(true),
		AutoPauseMemory: new(false),
		AutoResume:      new(true),
		Secure:          new(true),
		EnvVars:         map[string]string{"KEY": "value"},
		IAMTokens: map[string]IAMToken{
			"aws": {Audience: "sts.amazonaws.com", TokenType: "JWT-SVID"},
		},
		Network: &NetworkConfig{
			DenyOut:         []string{AllTraffic},
			MaskRequestHost: "proxy.internal",
		},
		VolumeMounts: []api.SandboxVolumeMount{{Name: "data", Path: "/mnt/data"}},
	})
	require.NoError(t, err)

	assert.Equal(t, "python", got["templateID"])
	assert.EqualValues(t, 600, got["timeout"])
	assert.Equal(t, true, got["autoPause"])
	assert.Equal(t, false, got["autoPauseMemory"], "autoPauseMemory was never sent before")
	assert.Equal(t, map[string]any{"enabled": true}, got["autoResume"])
	assert.Equal(t, true, got["secure"])
	assert.Equal(t, map[string]any{"KEY": "value"}, got["envVars"])

	// iam was absent from the SDK entirely.
	assert.Equal(t, map[string]any{
		"tokens": map[string]any{
			"aws": map[string]any{"audience": "sts.amazonaws.com", "tokenType": "JWT-SVID"},
		},
	}, got["iam"])

	assert.Equal(t, map[string]any{
		"denyOut":         []any{AllTraffic},
		"maskRequestHost": "proxy.internal",
	}, got["network"])

	assert.Equal(t, []any{map[string]any{"name": "data", "path": "/mnt/data"}}, got["volumeMounts"])
}

func TestCreateKeepsCallerMetadataAndManageBy(t *testing.T) {
	var got map[string]any
	svc := captureCreate(t, &got)

	caller := map[string]string{"owner": "alice"}
	_, err := svc.Create(context.Background(), CreateOptions{
		Metadata: caller,
		ManageBy: ManageByCodeBox,
	})
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"owner":             "alice",
		ManageByMetadataKey: ManageByCodeBox,
	}, got["metadata"])

	// The caller's own map must come back untouched.
	assert.Equal(t, map[string]string{"owner": "alice"}, caller)
}

func TestCreateRejectsATemplateTooOldToTalkTo(t *testing.T) {
	var killed bool
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			killed = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		created := createdSandbox()
		created["envdVersion"] = "0.0.9"
		writeJSON(t, w, http.StatusCreated, created)
	}))

	_, err := svc.Create(context.Background(), CreateOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too old")

	// Leaving the sandbox running would bill for something unusable.
	assert.True(t, killed, "a sandbox that cannot be driven should be shut down")
}

func TestPauseWithoutMemory(t *testing.T) {
	var got map[string]any
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusNoContent)
	}))

	// memory was never sent before, so a filesystem-only pause was impossible.
	require.NoError(t, svc.Pause(context.Background(), "sbx-1", PauseOptions{Memory: new(false)}))
	assert.Equal(t, false, got["memory"])
}

func TestPauseTreatsAnAlreadyPausedSandboxAsSuccess(t *testing.T) {
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))

	assert.NoError(t, svc.Pause(context.Background(), "sbx-1", PauseOptions{}),
		"the caller wanted the sandbox paused, and it is")
}

func TestKillMissingSandboxIsFalse(t *testing.T) {
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	ok, err := svc.Kill(context.Background(), "gone")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestListV2SendsFilters(t *testing.T) {
	var gotQuery string
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(t, w, http.StatusOK, []map[string]any{})
	}))

	_, err := svc.ListV2(context.Background(), ListV2Options{
		Metadata:        map[string]string{"owner": "alice"},
		State:           []State{"running"},
		Template:        "base",
		OrderDescending: true,
		Limit:           50,
	}).NextItems(context.Background())
	require.NoError(t, err)

	// None of these filters were reachable before.
	assert.Contains(t, gotQuery, "metadata=owner%3Dalice")
	assert.Contains(t, gotQuery, "state=running")
	assert.Contains(t, gotQuery, "template=base")
	assert.Contains(t, gotQuery, "order=desc")
	assert.Contains(t, gotQuery, "limit=50")
}

func TestListV2Paginates(t *testing.T) {
	var calls int
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("nextToken") == "" {
			w.Header().Set(transport.NextTokenHeader, "cursor-2")
			writeJSON(t, w, http.StatusOK, []map[string]any{listedSandbox("sbx-1")})
			return
		}
		writeJSON(t, w, http.StatusOK, []map[string]any{listedSandbox("sbx-2")})
	}))

	all, err := svc.ListV2(context.Background(), ListV2Options{}).All(context.Background())
	require.NoError(t, err)

	// The old paginator always reported a single page, so a team with more
	// sandboxes than one page silently lost the rest.
	require.Len(t, all, 2)
	assert.Equal(t, "sbx-1", all[0].SandboxID)
	assert.Equal(t, "sbx-2", all[1].SandboxID)
	assert.Equal(t, 2, calls)
}

func listedSandbox(id string) map[string]any {
	return map[string]any{
		"sandboxID":   id,
		"templateID":  "base",
		"clientID":    "c",
		"startedAt":   "2026-09-01T10:00:00Z",
		"endAt":       "2026-09-01T11:00:00Z",
		"state":       "running",
		"cpuCount":    2,
		"memoryMB":    512,
		"diskSizeMB":  1024,
		"envdVersion": "0.4.0",
	}
}

func TestUpdateNetworkReplacesThePolicy(t *testing.T) {
	var got map[string]any
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusNoContent)
	}))

	err := svc.UpdateNetwork(context.Background(), "sbx-1", NetworkUpdate{
		DenyOut:             []string{AllTraffic},
		AllowInternetAccess: new(false),
	})
	require.NoError(t, err)

	assert.Equal(t, []any{AllTraffic}, got["denyOut"])
	assert.Equal(t, false, got["allow_internet_access"])
}

func TestForkReportsEachOutcomeSeparately(t *testing.T) {
	svc := newTestService(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, http.StatusCreated, []map[string]any{
			{"sandbox": createdSandbox()},
			{"error": map[string]any{"code": 507, "message": "no capacity"}},
		})
	}))

	results, err := svc.Fork(context.Background(), "sbx-1", ForkOptions{Count: 2})
	require.NoError(t, err, "one failed fork is not a failure of the call")
	require.Len(t, results, 2)

	require.NotNil(t, results[0].Sandbox)
	assert.NoError(t, results[0].Err)

	assert.Nil(t, results[1].Sandbox)
	require.Error(t, results[1].Err)
	assert.Contains(t, results[1].Err.Error(), "no capacity")
}

// decodeBody reads a request's JSON body into target.
func decodeBody(t *testing.T, r *http.Request, target any) {
	t.Helper()
	require.NoError(t, json.NewDecoder(r.Body).Decode(target))
}
