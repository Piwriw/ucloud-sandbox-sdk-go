package sandbox

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	config ConnectionConfig
}

func NewClient(apiDomain, apiKey string, opts ...ClientOption) *Client {
	cfg := ConnectionConfig{
		Domain:          apiDomain,
		APIKey:          apiKey,
		InsecureSkipTLS: true,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.APIURL == "" {
		scheme := "https"
		if cfg.InsecureHTTP {
			scheme = "http"
		}
		cfg.APIURL = scheme + "://api." + apiDomain
	}
	if cfg.VolumeAPIURL == "" {
		cfg.VolumeAPIURL = cfg.APIURL
	}
	return &Client{config: cfg}
}

func (c *Client) newSandboxHTTPClient() *http.Client {
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		baseTransport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	}
	transport := baseTransport.Clone()
	transport.Proxy = http.ProxyFromEnvironment
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	transport.TLSClientConfig.InsecureSkipVerify = c.config.InsecureSkipTLS //nolint:gosec
	transport.DisableCompression = true

	return &http.Client{
		Timeout:   5 * time.Minute,
		Transport: transport,
	}
}

func (c *Client) newSandbox(sandboxID, sandboxDomain, envdVersion, envdAccessToken, trafficAccessToken string) *Sandbox {
	sbx := &Sandbox{
		ID:                 sandboxID,
		SandboxDomain:      sandboxDomain,
		client:             c,
		envdAccessToken:    envdAccessToken,
		trafficAccessToken: trafficAccessToken,
		envdVersion:        parseEnvdVersion(envdVersion),
		httpClient:         c.newSandboxHTTPClient(),
	}
	sbx.envdAPIURL = c.config.GetSandboxURL(sandboxID, sandboxDomain)
	rpc := sbx.newRPC()
	sbx.Commands = newCommands(sbx, rpc)
	sbx.Files = newFilesystem(sbx, rpc)
	sbx.Pty = newPty(sbx, rpc)
	return sbx
}

func (c *Client) CreateSandbox(ctx context.Context, opts ...SandboxOption) (*Sandbox, error) {
	cfg := &sandboxConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	applySandboxDefaults(cfg)
	reqBody := createSandboxRequest{
		TemplateID:          cfg.template,
		Timeout:             cfg.timeout,
		Metadata:            cfg.metadata,
		EnvVars:             cfg.envVars,
		AutoPause:           cfg.autoPause,
		Secure:              cfg.secure,
		AllowInternetAccess: cfg.allowInternetAccess,
		Network:             cfg.network,
		AutoResume:          toAutoResumeObject(cfg.autoResume),
		VolumeMounts:        cfg.volumeMounts,
		MCP:                 cfg.mcp,
	}
	var resp createSandboxResponse
	if err := c.doRequest(ctx, http.MethodPost, "/sandboxes", reqBody, &resp); err != nil {
		return nil, fmt.Errorf("create sandbox: %w", err)
	}
	if versionLessThan(parseEnvdVersion(resp.EnvdVersion), [3]int{0, 1, 0}) {
		_ = c.doRequest(ctx, http.MethodDelete, "/sandboxes/"+resp.SandboxID, nil, nil)
		return nil, &TemplateError{SandboxError{Message: "template too old, run `e2b template build`"}}
	}
	sandboxDomain := resp.Domain
	if sandboxDomain == "" || !strings.Contains(sandboxDomain, ".") {
		sandboxDomain = c.config.Domain
	}
	return c.newSandbox(resp.SandboxID, sandboxDomain, resp.EnvdVersion, resp.EnvdAccessToken, resp.TrafficAccessToken), nil
}

func (c *Client) ConnectSandbox(ctx context.Context, sandboxID string, timeoutSeconds ...int) (*Sandbox, error) {
	timeout := DefaultSandboxTimeout
	if len(timeoutSeconds) > 0 && timeoutSeconds[0] > 0 {
		timeout = timeoutSeconds[0]
	}
	body := map[string]any{"timeout": timeout}
	var resp connectSandboxResponse
	if err := c.doRequest(ctx, http.MethodPost, "/sandboxes/"+sandboxID+"/connect", body, &resp); err != nil {
		return nil, fmt.Errorf("connect sandbox %s: %w", sandboxID, err)
	}
	sandboxDomain := resp.Domain
	if sandboxDomain == "" || !strings.Contains(sandboxDomain, ".") {
		sandboxDomain = c.config.Domain
	}
	return c.newSandbox(sandboxID, sandboxDomain, resp.EnvdVersion, resp.EnvdAccessToken, resp.TrafficAccessToken), nil
}

func (c *Client) GetSandboxInfo(ctx context.Context, sandboxID string) (*SandboxInfo, error) {
	var out SandboxInfo
	if err := c.doRequest(ctx, http.MethodGet, "/sandboxes/"+sandboxID, nil, &out); err != nil {
		return nil, fmt.Errorf("get sandbox %s: %w", sandboxID, err)
	}
	return &out, nil
}

func (c *Client) KillSandbox(ctx context.Context, sandboxID string) (bool, error) {
	if err := c.doRequest(ctx, http.MethodDelete, "/sandboxes/"+sandboxID, nil, nil); err != nil {
		if _, ok := err.(*NotFoundError); ok {
			return false, nil
		}
		return false, fmt.Errorf("kill sandbox %s: %w", sandboxID, err)
	}
	return true, nil
}

func (c *Client) SetSandboxTimeout(ctx context.Context, sandboxID string, timeoutSeconds int) error {
	return c.doRequest(ctx, http.MethodPost, "/sandboxes/"+sandboxID+"/timeout", map[string]int{"timeout": timeoutSeconds}, nil)
}

func (c *Client) PauseSandbox(ctx context.Context, sandboxID string) error {
	err := c.doRequest(ctx, http.MethodPost, "/sandboxes/"+sandboxID+"/pause", nil, nil)
	if err != nil && isConflictError(err) {
		return nil
	}
	return err
}

func (c *Client) ListSandboxes(ctx context.Context, query *SandboxQuery) *Paginator[SandboxInfo] {
	return newPaginator(0, func(ctx context.Context, token string, limit int) ([]SandboxInfo, string, error) {
		path := "/v2/sandboxes"
		sep := "?"
		if token != "" {
			path += sep + "nextToken=" + url.QueryEscape(token)
			sep = "&"
		}
		if limit > 0 {
			path += sep + fmt.Sprintf("limit=%d", limit)
			sep = "&"
		}
		if query != nil {
			for k, v := range query.Metadata {
				path += sep + "metadata=" + url.QueryEscape(k) + "=" + url.QueryEscape(v)
				sep = "&"
			}
			if len(query.State) > 0 {
				stateStrs := make([]string, 0, len(query.State))
				for _, s := range query.State {
					stateStrs = append(stateStrs, url.QueryEscape(s))
				}
				path += sep + "state=" + strings.Join(stateStrs, ",")
				sep = "&"
			}
		}
		var sandboxes []SandboxInfo
		if err := c.doRequest(ctx, http.MethodGet, path, nil, &sandboxes); err != nil {
			return nil, "", err
		}
		return sandboxes, "", nil
	})
}

func (c *Client) ListSnapshots(ctx context.Context, sandboxID *string) *Paginator[SnapshotInfo] {
	return newPaginator(0, func(ctx context.Context, token string, limit int) ([]SnapshotInfo, string, error) {
		path := "/snapshots"
		sep := "?"
		if sandboxID != nil {
			path += sep + "sandbox_id=" + url.QueryEscape(*sandboxID)
			sep = "&"
		}
		if token != "" {
			path += sep + "next_token=" + url.QueryEscape(token)
		}
		var snapshots []SnapshotInfo
		if err := c.doRequest(ctx, http.MethodGet, path, nil, &snapshots); err != nil {
			return nil, "", err
		}
		return snapshots, "", nil
	})
}

func (c *Client) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	return c.doRequest(ctx, http.MethodDelete, "/templates/"+url.PathEscape(snapshotID), nil, nil)
}

// SnapshotExists checks whether a snapshot template still exists on the sandbox platform.
func (c *Client) SnapshotExists(ctx context.Context, snapshotID string) (bool, error) {
	paginator := c.ListSnapshots(ctx, nil)
	for paginator.HasNext() {
		items, err := paginator.NextItems(ctx)
		if err != nil {
			return false, fmt.Errorf("list snapshots: %w", err)
		}
		for _, s := range items {
			if s.SnapshotID == snapshotID {
				return true, nil
			}
		}
	}
	return false, nil
}

func (c *Client) ListTemplates(ctx context.Context) *Paginator[TemplateInfo] {
	return newPaginator(0, func(ctx context.Context, token string, limit int) ([]TemplateInfo, string, error) {
		var templates []TemplateInfo
		if err := c.doRequest(ctx, http.MethodGet, "/templates", nil, &templates); err != nil {
			return nil, "", err
		}
		return templates, "", nil
	})
}

func (c *Client) DeleteTemplate(ctx context.Context, templateID string) error {
	err := c.doRequest(ctx, http.MethodDelete, "/templates/"+url.PathEscape(templateID), nil, nil)
	if err != nil {
		return fmt.Errorf("delete template %s: %w", templateID, err)
	}
	return nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body, result any) error {
	fullURL := c.config.APIURL + path

	if body == nil && methodHasJSONBody(method) {
		body = map[string]any{}
	}

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("X-API-Key", c.config.APIKey)

	httpClient := c.newSandboxHTTPClient()
	if c.config.RequestTimeout > 0 {
		httpClient.Timeout = c.config.RequestTimeout
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapHTTPError(resp.StatusCode, string(respBody))
	}
	if result == nil || len(respBody) == 0 {
		return nil
	}
	if raw, ok := result.(*json.RawMessage); ok {
		*raw = append((*raw)[:0], respBody...)
		return nil
	}
	return json.Unmarshal(respBody, result)
}

func methodHasJSONBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}
