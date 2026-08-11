package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	defaultVolumeRequestTimeout = time.Minute
	defaultVolumeFileTimeout    = time.Hour
)

// VolumeInfo contains the public information for a persistent volume.
type VolumeInfo struct {
	VolumeID string `json:"volumeID"`
	Name     string `json:"name"`
}

// VolumeAndToken contains volume information and the token used for content operations.
type VolumeAndToken struct {
	VolumeInfo
	Token string `json:"token"`
}

// VolumeEntryType identifies a filesystem object stored in a volume.
type VolumeEntryType string

const (
	VolumeEntryTypeUnknown   VolumeEntryType = "unknown"
	VolumeEntryTypeFile      VolumeEntryType = "file"
	VolumeEntryTypeDirectory VolumeEntryType = "directory"
	VolumeEntryTypeSymlink   VolumeEntryType = "symlink"
)

// VolumeEntryStat contains metadata for a file or directory in a volume.
type VolumeEntryStat struct {
	Name   string          `json:"name"`
	Type   VolumeEntryType `json:"type"`
	Path   string          `json:"path"`
	Size   int64           `json:"size"`
	Mode   uint32          `json:"mode"`
	UID    uint32          `json:"uid"`
	GID    uint32          `json:"gid"`
	ATime  time.Time       `json:"atime"`
	MTime  time.Time       `json:"mtime"`
	CTime  time.Time       `json:"ctime"`
	Target *string         `json:"target,omitempty"`
}

// Volume is persistent storage that can be mounted into sandboxes.
type Volume struct {
	ID    string
	Name  string
	Token string

	client *Client
}

type volumeContentConfig struct {
	uid   *uint32
	gid   *uint32
	mode  *uint32
	force *bool
	depth *uint32
}

// VolumeContentOption configures volume directory and file operations.
type VolumeContentOption func(*volumeContentConfig)

// WithVolumeUID sets the owner user ID for a volume entry operation.
func WithVolumeUID(uid uint32) VolumeContentOption {
	return func(c *volumeContentConfig) { c.uid = &uid }
}

// WithVolumeGID sets the owner group ID for a volume entry operation.
func WithVolumeGID(gid uint32) VolumeContentOption {
	return func(c *volumeContentConfig) { c.gid = &gid }
}

// WithVolumeMode sets the Unix permission mode for a volume entry operation.
func WithVolumeMode(mode uint32) VolumeContentOption {
	return func(c *volumeContentConfig) { c.mode = &mode }
}

// WithVolumeForce controls parent creation for directories and overwrite behavior for files.
func WithVolumeForce(force bool) VolumeContentOption {
	return func(c *volumeContentConfig) { c.force = &force }
}

// WithVolumeDepth sets the recursive depth for Volume.List.
func WithVolumeDepth(depth uint32) VolumeContentOption {
	return func(c *volumeContentConfig) { c.depth = &depth }
}

func applyVolumeContentOptions(opts []VolumeContentOption) *volumeContentConfig {
	cfg := &volumeContentConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// CreateVolume creates a new team volume and returns it with its content token.
func (c *Client) CreateVolume(ctx context.Context, name string) (*Volume, error) {
	var out VolumeAndToken
	if err := c.doRequest(ctx, http.MethodPost, "/volumes", map[string]string{"name": name}, &out); err != nil {
		return nil, fmt.Errorf("create volume: %w", err)
	}
	return c.newVolume(out), nil
}

// ConnectVolume retrieves an existing volume and its content token.
func (c *Client) ConnectVolume(ctx context.Context, volumeID string) (*Volume, error) {
	info, err := c.GetVolumeInfo(ctx, volumeID)
	if err != nil {
		return nil, err
	}
	return c.newVolume(*info), nil
}

// GetVolumeInfo retrieves a volume and its content token.
func (c *Client) GetVolumeInfo(ctx context.Context, volumeID string) (*VolumeAndToken, error) {
	var out VolumeAndToken
	if err := c.doRequest(ctx, http.MethodGet, "/volumes/"+url.PathEscape(volumeID), nil, &out); err != nil {
		return nil, fmt.Errorf("get volume %s: %w", volumeID, err)
	}
	return &out, nil
}

// ListVolumes lists all team volumes.
func (c *Client) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	var out []VolumeInfo
	if err := c.doRequest(ctx, http.MethodGet, "/volumes", nil, &out); err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}
	if out == nil {
		return []VolumeInfo{}, nil
	}
	return out, nil
}

// DeleteVolume deletes a team volume. It returns false when the volume does not exist.
func (c *Client) DeleteVolume(ctx context.Context, volumeID string) (bool, error) {
	err := c.doRequest(ctx, http.MethodDelete, "/volumes/"+url.PathEscape(volumeID), nil, nil)
	if err == nil {
		return true, nil
	}
	var notFoundErr *NotFoundError
	if errors.As(err, &notFoundErr) {
		return false, nil
	}
	return false, fmt.Errorf("delete volume %s: %w", volumeID, err)
}

// DestroyVolume is an alias for DeleteVolume.
func (c *Client) DestroyVolume(ctx context.Context, volumeID string) (bool, error) {
	return c.DeleteVolume(ctx, volumeID)
}

func (c *Client) newVolume(info VolumeAndToken) *Volume {
	return &Volume{ID: info.VolumeID, Name: info.Name, Token: info.Token, client: c}
}

// Mount returns a sandbox volume mount for this volume.
func (v *Volume) Mount(path string) VolumeMount {
	return VolumeMount{Name: v.Name, Path: path}
}

// Destroy deletes the volume. It returns false when the volume no longer exists.
func (v *Volume) Destroy(ctx context.Context) (bool, error) {
	return v.client.DeleteVolume(ctx, v.ID)
}

// List returns the entries below path. Use WithVolumeDepth to recurse.
func (v *Volume) List(ctx context.Context, path string, opts ...VolumeContentOption) ([]VolumeEntryStat, error) {
	cfg := applyVolumeContentOptions(opts)
	query := url.Values{"path": []string{path}}
	if cfg.depth != nil {
		query.Set("depth", strconv.FormatUint(uint64(*cfg.depth), 10))
	}
	var out []VolumeEntryStat
	if err := v.doContentJSON(ctx, http.MethodGet, "dir", query, nil, &out, false); err != nil {
		return nil, fmt.Errorf("list volume path %s: %w", path, err)
	}
	if out == nil {
		return []VolumeEntryStat{}, nil
	}
	return out, nil
}

// MakeDir creates a directory. Metadata and parent creation can be configured with volume content options.
func (v *Volume) MakeDir(ctx context.Context, path string, opts ...VolumeContentOption) (*VolumeEntryStat, error) {
	cfg := applyVolumeContentOptions(opts)
	query := volumeEntryQuery(path, cfg, true)
	var out VolumeEntryStat
	if err := v.doContentJSON(ctx, http.MethodPost, "dir", query, nil, &out, false); err != nil {
		return nil, fmt.Errorf("make volume directory %s: %w", path, err)
	}
	return &out, nil
}

// Exists reports whether path exists in the volume.
func (v *Volume) Exists(ctx context.Context, path string) (bool, error) {
	_, err := v.GetInfo(ctx, path)
	if err == nil {
		return true, nil
	}
	var notFoundErr *NotFoundError
	if errors.As(err, &notFoundErr) {
		return false, nil
	}
	return false, err
}

// GetInfo returns metadata for a file or directory.
func (v *Volume) GetInfo(ctx context.Context, path string) (*VolumeEntryStat, error) {
	query := url.Values{"path": []string{path}}
	var out VolumeEntryStat
	if err := v.doContentJSON(ctx, http.MethodGet, "path", query, nil, &out, false); err != nil {
		return nil, fmt.Errorf("get volume path %s: %w", path, err)
	}
	return &out, nil
}

// UpdateMetadata updates the UID, GID, or mode of a file or directory.
func (v *Volume) UpdateMetadata(ctx context.Context, path string, opts ...VolumeContentOption) (*VolumeEntryStat, error) {
	cfg := applyVolumeContentOptions(opts)
	body := struct {
		UID  *uint32 `json:"uid,omitempty"`
		GID  *uint32 `json:"gid,omitempty"`
		Mode *uint32 `json:"mode,omitempty"`
	}{UID: cfg.uid, GID: cfg.gid, Mode: cfg.mode}
	query := url.Values{"path": []string{path}}
	var out VolumeEntryStat
	if err := v.doContentJSON(ctx, http.MethodPatch, "path", query, body, &out, false); err != nil {
		return nil, fmt.Errorf("update volume path %s: %w", path, err)
	}
	return &out, nil
}

// ReadFile reads an entire file into memory.
func (v *Volume) ReadFile(ctx context.Context, path string) ([]byte, error) {
	reader, err := v.OpenFile(ctx, path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read volume file %s: %w", path, err)
	}
	return data, nil
}

// ReadFileText reads an entire UTF-8 file as a string.
func (v *Volume) ReadFileText(ctx context.Context, path string) (string, error) {
	data, err := v.ReadFile(ctx, path)
	return string(data), err
}

// OpenFile opens a streaming response for a file. The caller must close it.
func (v *Volume) OpenFile(ctx context.Context, path string) (io.ReadCloser, error) {
	query := url.Values{"path": []string{path}}
	resp, err := v.doContentRequest(ctx, http.MethodGet, "file", query, nil, "", true)
	if err != nil {
		return nil, fmt.Errorf("open volume file %s: %w", path, err)
	}
	return resp.Body, nil
}

// WriteFile writes a string, byte slice, or io.Reader to a file.
func (v *Volume) WriteFile(ctx context.Context, path string, data any, opts ...VolumeContentOption) (*VolumeEntryStat, error) {
	var reader io.Reader
	switch value := data.(type) {
	case string:
		reader = bytes.NewBufferString(value)
	case []byte:
		reader = bytes.NewReader(value)
	case io.Reader:
		reader = value
	default:
		return nil, fmt.Errorf("unsupported volume file data type: %T", data)
	}
	return v.WriteFileFrom(ctx, path, reader, opts...)
}

// WriteFileFrom streams data from reader into a volume file.
func (v *Volume) WriteFileFrom(ctx context.Context, path string, reader io.Reader, opts ...VolumeContentOption) (*VolumeEntryStat, error) {
	cfg := applyVolumeContentOptions(opts)
	query := volumeEntryQuery(path, cfg, true)
	resp, err := v.doContentRequest(ctx, http.MethodPut, "file", query, reader, "application/octet-stream", true)
	if err != nil {
		return nil, fmt.Errorf("write volume file %s: %w", path, err)
	}
	defer resp.Body.Close()
	var out VolumeEntryStat
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode written volume file %s: %w", path, err)
	}
	return &out, nil
}

// Remove recursively removes a file or directory.
func (v *Volume) Remove(ctx context.Context, path string) error {
	query := url.Values{"path": []string{path}}
	if err := v.doContentJSON(ctx, http.MethodDelete, "path", query, nil, nil, false); err != nil {
		return fmt.Errorf("remove volume path %s: %w", path, err)
	}
	return nil
}

func volumeEntryQuery(path string, cfg *volumeContentConfig, includeForce bool) url.Values {
	query := url.Values{"path": []string{path}}
	if cfg.uid != nil {
		query.Set("uid", strconv.FormatUint(uint64(*cfg.uid), 10))
	}
	if cfg.gid != nil {
		query.Set("gid", strconv.FormatUint(uint64(*cfg.gid), 10))
	}
	if cfg.mode != nil {
		query.Set("mode", strconv.FormatUint(uint64(*cfg.mode), 10))
	}
	if includeForce && cfg.force != nil {
		query.Set("force", strconv.FormatBool(*cfg.force))
	}
	return query
}

func (v *Volume) doContentJSON(ctx context.Context, method, resource string, query url.Values, body, result any, file bool) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	contentType := ""
	if body != nil {
		contentType = "application/json"
	}
	resp, err := v.doContentRequest(ctx, method, resource, query, reader, contentType, file)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if result == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return err
	}
	return nil
}

func (v *Volume) doContentRequest(ctx context.Context, method, resource string, query url.Values, body io.Reader, contentType string, file bool) (*http.Response, error) {
	if v.client == nil {
		return nil, &SandboxError{Message: "volume is not connected to a client"}
	}
	if v.Token == "" {
		return nil, &AuthenticationError{SandboxError{Message: "volume token is required for content operations"}}
	}
	endpoint := v.client.config.VolumeAPIURL + "/volumecontent/" + url.PathEscape(v.ID) + "/" + resource
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	for key, value := range v.client.config.Headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Authorization", "Bearer "+v.Token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	timeout := defaultVolumeRequestTimeout
	if file {
		timeout = defaultVolumeFileTimeout
	}
	if v.client.config.RequestTimeout > 0 {
		timeout = v.client.config.RequestTimeout
	}
	httpClient := v.client.newSandboxHTTPClient()
	httpClient.Timeout = timeout
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}
	defer resp.Body.Close()
	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}
	return nil, mapHTTPError(resp.StatusCode, string(respBody))
}
