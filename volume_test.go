package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestVolumeManagement(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("X-API-Key"), "api-key"; got != want {
			t.Errorf("X-API-Key = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("X-Test-Header"), "test-value"; got != want {
			t.Errorf("X-Test-Header = %q, want %q", got, want)
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/volumes":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode create body: %v", err)
			}
			if got, want := body["name"], "workspace"; got != want {
				t.Errorf("volume name = %q, want %q", got, want)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"volumeID":"vol-1","name":"workspace","token":"token-1"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/volumes/vol-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"volumeID":"vol-1","name":"workspace","token":"token-1"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/volumes":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `[{"volumeID":"vol-1","name":"workspace"}]`)
		case r.Method == http.MethodDelete && r.URL.Path == "/volumes/vol-1":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodDelete && r.URL.Path == "/volumes/missing":
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"code":"not_found","message":"missing"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(
		"example.com",
		"api-key",
		WithAPIURL(server.URL),
		WithHeaders(map[string]string{"X-Test-Header": "test-value"}),
	)
	ctx := context.Background()

	volume, err := client.CreateVolume(ctx, "workspace")
	if err != nil {
		t.Fatalf("CreateVolume: %v", err)
	}
	if got, want := volume.ID, "vol-1"; got != want {
		t.Fatalf("volume ID = %q, want %q", got, want)
	}
	if got, want := volume.Token, "token-1"; got != want {
		t.Fatalf("volume token = %q, want %q", got, want)
	}
	if got, want := volume.Mount("/workspace"), (VolumeMount{Name: "workspace", Path: "/workspace"}); got != want {
		t.Fatalf("mount = %#v, want %#v", got, want)
	}

	connected, err := client.ConnectVolume(ctx, "vol-1")
	if err != nil {
		t.Fatalf("ConnectVolume: %v", err)
	}
	if got, want := connected.Name, "workspace"; got != want {
		t.Fatalf("connected volume name = %q, want %q", got, want)
	}

	volumes, err := client.ListVolumes(ctx)
	if err != nil {
		t.Fatalf("ListVolumes: %v", err)
	}
	if got, want := volumes, []VolumeInfo{{VolumeID: "vol-1", Name: "workspace"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("volumes = %#v, want %#v", got, want)
	}

	deleted, err := client.DeleteVolume(ctx, "vol-1")
	if err != nil || !deleted {
		t.Fatalf("DeleteVolume = %v, %v; want true, nil", deleted, err)
	}
	deleted, err = client.DeleteVolume(ctx, "missing")
	if err != nil || deleted {
		t.Fatalf("DeleteVolume missing = %v, %v; want false, nil", deleted, err)
	}
}

func TestVolumeContent(t *testing.T) {
	t.Parallel()

	const statJSON = `{"name":"hello.txt","type":"file","path":"/hello.txt","size":5,"mode":420,"uid":1000,"gid":1000,"atime":"2026-08-11T01:02:03Z","mtime":"2026-08-11T01:02:03Z","ctime":"2026-08-11T01:02:03Z"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer volume-token"; got != want {
			t.Errorf("Authorization = %q, want %q", got, want)
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /volumecontent/vol-1/path":
			if r.URL.Query().Get("path") == "/missing" {
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"code":"not_found","message":"missing"}`)
				return
			}
			_, _ = io.WriteString(w, statJSON)
		case "GET /volumecontent/vol-1/dir":
			if got, want := r.URL.Query().Get("depth"), "2"; got != want {
				t.Errorf("depth = %q, want %q", got, want)
			}
			_, _ = io.WriteString(w, "["+statJSON+"]")
		case "POST /volumecontent/vol-1/dir":
			assertVolumeMetadataQuery(t, r, true)
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, statJSON)
		case "PATCH /volumecontent/vol-1/path":
			var body map[string]uint32
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode metadata body: %v", err)
			}
			if got, want := body, map[string]uint32{"uid": 1000, "gid": 1000, "mode": 420}; !reflect.DeepEqual(got, want) {
				t.Errorf("metadata body = %#v, want %#v", got, want)
			}
			_, _ = io.WriteString(w, statJSON)
		case "GET /volumecontent/vol-1/file":
			_, _ = io.WriteString(w, "hello")
		case "PUT /volumecontent/vol-1/file":
			assertVolumeMetadataQuery(t, r, true)
			if got, want := r.Header.Get("Content-Type"), "application/octet-stream"; got != want {
				t.Errorf("Content-Type = %q, want %q", got, want)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read upload: %v", err)
			}
			if got, want := string(body), "hello"; got != want {
				t.Errorf("upload body = %q, want %q", got, want)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, statJSON)
		case "DELETE /volumecontent/vol-1/path":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient("example.com", "api-key", WithAPIURL(server.URL))
	volume := &Volume{ID: "vol-1", Name: "workspace", Token: "volume-token", client: client}
	ctx := context.Background()
	metadataOpts := []VolumeContentOption{
		WithVolumeUID(1000),
		WithVolumeGID(1000),
		WithVolumeMode(0o644),
		WithVolumeForce(true),
	}

	entries, err := volume.List(ctx, "/", WithVolumeDepth(2))
	if err != nil || len(entries) != 1 {
		t.Fatalf("List = %#v, %v", entries, err)
	}
	if _, err := volume.MakeDir(ctx, "/dir", metadataOpts...); err != nil {
		t.Fatalf("MakeDir: %v", err)
	}
	info, err := volume.GetInfo(ctx, "/hello.txt")
	if err != nil {
		t.Fatalf("GetInfo: %v", err)
	}
	if got, want := info.Type, VolumeEntryTypeFile; got != want {
		t.Fatalf("entry type = %q, want %q", got, want)
	}
	if _, err := volume.UpdateMetadata(ctx, "/hello.txt", metadataOpts...); err != nil {
		t.Fatalf("UpdateMetadata: %v", err)
	}
	if _, err := volume.WriteFile(ctx, "/hello.txt", "hello", metadataOpts...); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	content, err := volume.ReadFileText(ctx, "/hello.txt")
	if err != nil || content != "hello" {
		t.Fatalf("ReadFileText = %q, %v; want hello, nil", content, err)
	}
	exists, err := volume.Exists(ctx, "/missing")
	if err != nil || exists {
		t.Fatalf("Exists missing = %v, %v; want false, nil", exists, err)
	}
	if err := volume.Remove(ctx, "/hello.txt"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
}

func TestVolumeContentRequiresToken(t *testing.T) {
	t.Parallel()

	volume := &Volume{ID: "vol-1", client: NewClient("example.com", "api-key")}
	_, err := volume.GetInfo(context.Background(), "/hello.txt")
	var authErr *AuthenticationError
	if !errors.As(err, &authErr) {
		t.Fatalf("GetInfo error = %v, want AuthenticationError", err)
	}
}

func assertVolumeMetadataQuery(t *testing.T, r *http.Request, includeForce bool) {
	t.Helper()
	wants := map[string]string{
		"uid":  "1000",
		"gid":  "1000",
		"mode": "420",
	}
	if includeForce {
		wants["force"] = "true"
	}
	for key, want := range wants {
		if got := r.URL.Query().Get(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}
