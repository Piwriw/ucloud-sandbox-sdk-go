package sandbox

import "testing"

func TestWithInsecureHTTP(t *testing.T) {
	client := NewClient("example.com", "key", WithInsecureHTTP(true))

	if got, want := client.config.APIURL, "http://api.example.com"; got != want {
		t.Fatalf("APIURL = %q, want %q", got, want)
	}

	sbx := client.newSandbox("sandbox-id", "example.com", "0.1.0", "", "")
	if got, want := sbx.envdAPIURL, "http://49983-sandbox-id.example.com"; got != want {
		t.Fatalf("envdAPIURL = %q, want %q", got, want)
	}
	if got, want := sbx.Files.rpc.baseURL, sbx.envdAPIURL; got != want {
		t.Fatalf("filesystem RPC baseURL = %q, want %q", got, want)
	}
	if got, want := sbx.Commands.rpc.baseURL, sbx.envdAPIURL; got != want {
		t.Fatalf("process RPC baseURL = %q, want %q", got, want)
	}
	if got, want := sbx.GetMCPURL(), "http://50005-sandbox-id.example.com/mcp"; got != want {
		t.Fatalf("MCP URL = %q, want %q", got, want)
	}
}

func TestWithInsecureHTTPDisabledByDefault(t *testing.T) {
	client := NewClient("example.com", "key")

	if got, want := client.config.APIURL, "https://api.example.com"; got != want {
		t.Fatalf("APIURL = %q, want %q", got, want)
	}
	if got, want := client.config.GetSandboxURL("sandbox-id", "example.com"), "https://49983-sandbox-id.example.com"; got != want {
		t.Fatalf("sandbox URL = %q, want %q", got, want)
	}
}

func TestWithAPIURLOverridesInsecureHTTPDefault(t *testing.T) {
	client := NewClient(
		"example.com",
		"key",
		WithInsecureHTTP(true),
		WithAPIURL("https://control.example.test"),
	)

	if got, want := client.config.APIURL, "https://control.example.test"; got != want {
		t.Fatalf("APIURL = %q, want %q", got, want)
	}
}
