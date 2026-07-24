package sandbox

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
)

type Sandbox struct {
	ID            string
	SandboxDomain string

	Commands *Commands
	Files    *Filesystem
	Pty      *Pty

	client             *Client
	envdAPIURL         string
	envdAccessToken    string
	trafficAccessToken string
	envdVersion        [3]int
	httpClient         *http.Client
	mcpToken           string
	mcpTokenMu         sync.Mutex
}

func (s *Sandbox) Kill(ctx context.Context) (bool, error) {
	return s.client.KillSandbox(ctx, s.ID)
}
func (s *Sandbox) SetTimeout(ctx context.Context, timeoutSeconds int) error {
	return s.client.SetSandboxTimeout(ctx, s.ID, timeoutSeconds)
}
func (s *Sandbox) GetInfo(ctx context.Context) (*SandboxInfo, error) {
	return s.client.GetSandboxInfo(ctx, s.ID)
}

func (s *Sandbox) GetMetrics(ctx context.Context, opts ...MetricsOption) ([]SandboxMetrics, error) {
	if versionLessThan(s.envdVersion, envdVersionMetrics) {
		return nil, &SandboxError{Message: "metrics not supported, rebuild your template"}
	}
	if versionLessThan(s.envdVersion, envdVersionDiskMetrics) {
		log.Println("[E2B] Disk metrics not supported, rebuild template for disk metrics.")
	}
	cfg := &metricsConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	path := "/sandboxes/" + s.ID + "/metrics"
	sep := "?"
	if !cfg.start.IsZero() {
		path += sep + fmt.Sprintf("start=%d", cfg.start.Unix())
		sep = "&"
	}
	if !cfg.end.IsZero() {
		path += sep + fmt.Sprintf("end=%d", cfg.end.Unix())
	}
	var metrics []SandboxMetrics
	if err := s.client.doRequest(ctx, http.MethodGet, path, nil, &metrics); err != nil {
		return nil, err
	}
	if metrics == nil {
		return []SandboxMetrics{}, nil
	}
	return metrics, nil
}

func (s *Sandbox) Pause(ctx context.Context) error {
	return s.client.PauseSandbox(ctx, s.ID)
}

func (s *Sandbox) IsRunning(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.envdAPIURL+"/health", nil)
	if err != nil {
		return false, err
	}
	s.setSandboxHeaders(req)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil || isTimeoutError(err) {
			return false, &TimeoutError{SandboxError: SandboxError{Message: "health check timed out", Cause: err}}
		}
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusBadGateway {
		return false, nil
	}
	return true, nil
}

func (s *Sandbox) GetHost(port int) string {
	return s.client.config.GetHost(s.ID, s.SandboxDomain, port)
}

func (s *Sandbox) CreateSnapshot(ctx context.Context) (*SnapshotInfo, error) {
	var resp snapshotResponse
	if err := s.client.doRequest(ctx, http.MethodPost, "/sandboxes/"+s.ID+"/snapshots", snapshotRequest{}, &resp); err != nil {
		return nil, err
	}
	return &SnapshotInfo{SnapshotID: resp.SnapshotID}, nil
}

func (s *Sandbox) DownloadURL(path string, opts ...FileURLOption) string {
	cfg := &fileURLConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return s.buildSignedURL(path, "read", cfg)
}

func (s *Sandbox) UploadURL(path string, opts ...FileURLOption) string {
	cfg := &fileURLConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return s.buildSignedURL(path, "write", cfg)
}

func (s *Sandbox) buildSignedURL(path, operation string, cfg *fileURLConfig) string {
	user := s.resolveUsername(cfg.user)
	params := url.Values{}
	params.Set("path", path)
	if user != "" {
		params.Set("username", user)
	}
	if s.envdAccessToken != "" {
		var expSec *int
		if cfg.expiration > 0 {
			expSec = &cfg.expiration
		}
		sig, exp, err := getSignature(path, operation, user, s.envdAccessToken, expSec)
		if err == nil {
			params.Set("signature", sig)
			if exp != nil {
				params.Set("signature_expiration", fmt.Sprintf("%d", *exp))
			}
		}
	}
	return s.envdAPIURL + "/files?" + params.Encode()
}

func (s *Sandbox) GetMCPURL() string {
	scheme := "https"
	if s.client.config.Debug || s.client.config.InsecureHTTP {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/mcp", scheme, s.GetHost(MCPPort))
}

func (s *Sandbox) GetMCPToken(ctx context.Context) (string, error) {
	s.mcpTokenMu.Lock()
	defer s.mcpTokenMu.Unlock()
	if s.mcpToken != "" {
		return s.mcpToken, nil
	}
	token, err := s.Files.Read(ctx, "/etc/mcp-gateway/.token", WithFsUser("root"))
	if err != nil {
		return "", err
	}
	s.mcpToken = token
	return token, nil
}

func (s *Sandbox) resolveUsername(user string) string {
	if user != "" {
		return user
	}
	if versionLessThan(s.envdVersion, envdVersionDefaultUser) {
		return "user"
	}
	return ""
}

func buildAuthHeader(user string) map[string]string {
	if user == "" {
		return nil
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(user + ":"))
	return map[string]string{"Authorization": "Basic " + encoded}
}

func (s *Sandbox) supportsStdin() bool {
	return !versionLessThan(s.envdVersion, envdVersionStdin)
}
func (s *Sandbox) supportsRecursiveWatch() bool {
	return !versionLessThan(s.envdVersion, envdVersionRecursiveWatch)
}

func (s *Sandbox) newRPC() *RPC {
	return NewRPC(s.envdAPIURL, s.httpClient, map[string]string{
		"X-Access-Token":           s.envdAccessToken,
		"E2B-Traffic-Access-Token": s.trafficAccessToken,
		"E2b-Sandbox-Id":           s.ID,
		"E2b-Sandbox-Port":         fmt.Sprintf("%d", EnvdPort),
		"Keepalive-Ping-Interval":  fmt.Sprintf("%d", KeepalivePingIntervalSec),
	})
}

func (s *Sandbox) setSandboxHeaders(req *http.Request) {
	s.setSandboxHeadersWithPort(req, EnvdPort)
}

func (s *Sandbox) setSandboxHeadersWithPort(req *http.Request, port int) {
	if s.envdAccessToken != "" {
		req.Header.Set("X-Access-Token", s.envdAccessToken)
	} else if s.client.config.APIKey != "" {
		req.Header.Set("X-API-Key", s.client.config.APIKey)
	}
	if s.trafficAccessToken != "" {
		req.Header.Set("E2B-Traffic-Access-Token", s.trafficAccessToken)
	}
	req.Header.Set("E2b-Sandbox-Id", s.ID)
	req.Header.Set("E2b-Sandbox-Port", fmt.Sprintf("%d", port))
}
