package sandbox

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

const proxyCoverageHelper = "UCLOUD_SANDBOX_SDK_PROXY_COVERAGE_HELPER"

func TestSandboxHTTPClientPreservesDefaultTransportAndProxy(t *testing.T) {
	client := NewClient("example.invalid", "key")
	httpClient := client.newSandboxHTTPClient()
	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport = %T, want *http.Transport", httpClient.Transport)
	}
	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Fatal("http.DefaultTransport is not *http.Transport")
	}
	if transport == defaultTransport {
		t.Fatal("sandbox transport must clone, not mutate, http.DefaultTransport")
	}
	if transport.Proxy == nil {
		t.Fatal("sandbox transport must preserve ProxyFromEnvironment")
	}
	if got, want := transport.MaxIdleConns, defaultTransport.MaxIdleConns; got != want {
		t.Fatalf("MaxIdleConns = %d, want cloned default %d", got, want)
	}
	if !transport.DisableCompression {
		t.Fatal("sandbox transport must keep response compression disabled")
	}
	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("sandbox transport must preserve the configured TLS behavior")
	}
}

func TestEnvironmentProxyCoversControlRPCFilesAndStream(t *testing.T) {
	if os.Getenv(proxyCoverageHelper) == "1" {
		runProxyCoverageHelper(t)
		return
	}

	var mu sync.Mutex
	seen := make(map[string]int)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		kind := ""
		switch r.URL.Path {
		case "/control":
			kind = "control"
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"ok":true}`)
		case "/test.Service/Unary":
			kind = "rpc"
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"value":"unary"}`)
		case "/files":
			kind = "files"
			_, _ = io.WriteString(w, "file-through-proxy")
		case "/test.Service/Stream":
			kind = "stream"
			w.Header().Set("Content-Type", "application/connect+json")
			_, _ = w.Write(encodeEnvelope([]byte(`{"value":"stream"}`)))
		default:
			http.Error(w, "unexpected proxy path: "+r.URL.Path, http.StatusNotFound)
			return
		}
		mu.Lock()
		seen[kind]++
		mu.Unlock()
	}))
	defer proxy.Close()

	command := exec.Command(os.Args[0], "-test.run=^TestEnvironmentProxyCoversControlRPCFilesAndStream$")
	command.Env = append(withoutProxyEnvironment(os.Environ()),
		proxyCoverageHelper+"=1",
		"HTTP_PROXY="+proxy.URL,
		"HTTPS_PROXY="+proxy.URL,
		"NO_PROXY=",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("proxy coverage helper failed: %v\n%s", err, output)
	}

	mu.Lock()
	defer mu.Unlock()
	for _, kind := range []string{"control", "rpc", "files", "stream"} {
		if got := seen[kind]; got != 1 {
			t.Fatalf("%s requests through proxy = %d, want 1; all=%v", kind, got, seen)
		}
	}
}

func runProxyCoverageHelper(t *testing.T) {
	ctx := context.Background()
	client := NewClient("example.invalid", "key", WithInsecureHTTP(true))
	var control struct {
		OK bool `json:"ok"`
	}
	if err := client.doRequest(ctx, http.MethodGet, "/control", nil, &control); err != nil {
		t.Fatalf("control plane request: %v", err)
	}
	if !control.OK {
		t.Fatal("control plane response did not traverse the test proxy")
	}

	sandbox := client.newSandbox("sandbox-id", "example.invalid", "0.1.0", "envd-token", "")
	var unary struct {
		Value string `json:"value"`
	}
	if err := sandbox.Commands.rpc.CallUnary(ctx, "test.Service", "Unary", map[string]string{"ping": "pong"}, &unary); err != nil {
		t.Fatalf("RPC request: %v", err)
	}
	if unary.Value != "unary" {
		t.Fatalf("RPC response = %q, want unary", unary.Value)
	}

	file, err := sandbox.Files.Read(ctx, "/tmp/file.txt")
	if err != nil {
		t.Fatalf("file request: %v", err)
	}
	if file != "file-through-proxy" {
		t.Fatalf("file response = %q", file)
	}

	stream, err := sandbox.Commands.rpc.CallServerStream(ctx, "test.Service", "Stream", map[string]string{"ping": "pong"}, 1_000)
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer stream.Close()
	var streamed struct {
		Value string `json:"value"`
	}
	if err := stream.Next(&streamed); err != nil {
		t.Fatalf("stream response: %v", err)
	}
	if streamed.Value != "stream" {
		t.Fatalf("stream response = %q, want stream", streamed.Value)
	}
}

func withoutProxyEnvironment(environment []string) []string {
	blocked := map[string]struct{}{
		"all_proxy": {}, "http_proxy": {}, "https_proxy": {}, "no_proxy": {},
	}
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, _ := strings.Cut(entry, "=")
		if _, found := blocked[strings.ToLower(key)]; found {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func ExampleClient_environmentProxy() {
	fmt.Println("HTTP_PROXY, HTTPS_PROXY, and NO_PROXY are honored by all SDK requests")
	// Output: HTTP_PROXY, HTTPS_PROXY, and NO_PROXY are honored by all SDK requests
}
