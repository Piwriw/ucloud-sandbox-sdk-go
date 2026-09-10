package sandbox

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type TemplateBuildStatus string

const (
	BuildStatusBuilding TemplateBuildStatus = "building"
	BuildStatusWaiting  TemplateBuildStatus = "waiting"
	BuildStatusReady    TemplateBuildStatus = "ready"
	BuildStatusError    TemplateBuildStatus = "error"
	BuildStatusUploaded TemplateBuildStatus = "uploaded"
)

type BuildInfo struct {
	TemplateID string   `json:"templateId"`
	BuildID    string   `json:"buildId"`
	Name       string   `json:"name"`
	Tags       []string `json:"tags,omitempty"`
}

type BuildStatusResponse struct {
	BuildID    string              `json:"buildId"`
	TemplateID string              `json:"templateId"`
	Status     TemplateBuildStatus `json:"status"`
	Logs       []string            `json:"logs"`
	LogEntries []LogEntry          `json:"logEntries,omitempty"`
	Reason     *BuildStatusReason  `json:"reason,omitempty"`
}

type BuildStatusReason struct {
	Message    string     `json:"message"`
	Step       string     `json:"step,omitempty"`
	LogEntries []LogEntry `json:"logEntries,omitempty"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

func (e LogEntry) String() string { return e.Message }

type buildConfig struct {
	tags              []string
	cpuCount          int
	memoryMB          int
	skipCache         bool
	publish           bool
	onBuildLogs       func(LogEntry)
	fromImageRegistry *RegistryConfig
}

type BuildOption func(*buildConfig)

func WithBuildTags(tags []string) BuildOption {
	return func(c *buildConfig) { c.tags = tags }
}

func WithBuildCPUCount(count int) BuildOption {
	return func(c *buildConfig) { c.cpuCount = count }
}

func WithBuildMemoryMB(mb int) BuildOption {
	return func(c *buildConfig) { c.memoryMB = mb }
}

func WithBuildSkipCache(skip bool) BuildOption {
	return func(c *buildConfig) { c.skipCache = skip }
}

func WithOnBuildLogs(fn func(LogEntry)) BuildOption {
	return func(c *buildConfig) { c.onBuildLogs = fn }
}

func WithPublishTemplate() BuildOption {
	return func(c *buildConfig) { c.publish = true }
}

// WithBuildFromImageRegistry sets credentials for pulling the base image from a
// private registry. The registry type defaults to "registry".
func WithBuildFromImageRegistry(username, password string) BuildOption {
	return func(c *buildConfig) {
		c.fromImageRegistry = &RegistryConfig{
			Type:     "registry",
			Username: username,
			Password: password,
		}
	}
}

func DefaultBuildLogger() func(LogEntry) {
	return DefaultBuildLoggerWithLevel("info")
}

func DefaultBuildLoggerWithLevel(minLevel string) func(LogEntry) {
	levelOrder := map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}
	minOrd, ok := levelOrder[minLevel]
	if !ok {
		minOrd = levelOrder["info"]
	}
	start := time.Now()
	return func(entry LogEntry) {
		entryOrd, ok := levelOrder[entry.Level]
		if !ok {
			entryOrd = levelOrder["info"]
		}
		if entryOrd < minOrd {
			return
		}
		elapsed := time.Since(start).Seconds()
		ts := entry.Timestamp.Format("15:04:05")
		if ts == "00:00:00" {
			ts = time.Now().Format("15:04:05")
		}
		fmt.Fprintf(os.Stderr, "%5.1fs | %s %-5s %s\n",
			elapsed, ts, strings.ToUpper(entry.Level), entry.Message)
	}
}

func applyBuildOpts(opts []BuildOption) *buildConfig {
	cfg := &buildConfig{
		cpuCount: 2,
		memoryMB: 1024,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func (c *Client) BuildTemplate(ctx context.Context, template *TemplateBuilder, name string, opts ...BuildOption) (*BuildInfo, error) {
	if template.err != nil {
		return nil, &TemplateError{SandboxError{Message: template.err.Error(), Cause: template.err}}
	}

	cfg := applyBuildOpts(opts)
	steps, err := template.prepareSteps()
	if err != nil {
		return nil, &TemplateError{SandboxError{Message: err.Error(), Cause: err}}
	}
	data := template.serialize()
	data.Steps = steps
	if cfg.skipCache {
		data.Force = true
	}
	if cfg.fromImageRegistry != nil {
		data.FromImageRegistry = cfg.fromImageRegistry
	}

	emitBuildLog(cfg, LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   fmt.Sprintf("Requesting build for template: %s", name),
	})

	var createResp requestBuildResponse
	if err := c.doRequest(ctx, http.MethodPost, "/v3/templates", requestBuildRequest{
		Name:     name,
		Tags:     cfg.tags,
		CPUCount: cfg.cpuCount,
		MemoryMB: cfg.memoryMB,
	}, &createResp); err != nil {
		return nil, err
	}

	emitBuildLog(cfg, LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   fmt.Sprintf("Template created with ID: %s, Build ID: %s", createResp.TemplateID, createResp.BuildID),
	})
	emitBuildLog(cfg, LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Message:   "Starting building...",
	})

	if err := c.uploadTemplateFiles(ctx, createResp.TemplateID, template, steps); err != nil {
		return nil, err
	}

	triggerPath := fmt.Sprintf("/v2/templates/%s/builds/%s", createResp.TemplateID, createResp.BuildID)
	if err := c.doRequest(ctx, http.MethodPost, triggerPath, data, nil); err != nil {
		return nil, err
	}

	tags := cfg.tags
	if len(createResp.Tags) > 0 {
		tags = createResp.Tags
	}

	info := &BuildInfo{
		TemplateID: createResp.TemplateID,
		BuildID:    createResp.BuildID,
		Name:       name,
		Tags:       tags,
	}

	if _, err := c.WaitForBuild(ctx, createResp.TemplateID, createResp.BuildID, opts...); err != nil {
		return info, err
	}

	if cfg.publish {
		emitBuildLog(cfg, LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "Publishing template...",
		})
		if err := c.PublishTemplate(ctx, createResp.TemplateID); err != nil {
			return info, err
		}
		emitBuildLog(cfg, LogEntry{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   "Template published",
		})
	}

	return info, nil
}

func emitBuildLog(cfg *buildConfig, entry LogEntry) {
	if cfg.onBuildLogs != nil {
		cfg.onBuildLogs(entry)
	}
}

func (c *Client) GetBuildStatus(ctx context.Context, templateID, buildID string, logsOffset int) (*BuildStatusResponse, error) {
	path := fmt.Sprintf("/templates/%s/builds/%s/status?logsOffset=%d", templateID, buildID, logsOffset)

	var resp buildStatusAPIResponse
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return &BuildStatusResponse{
		BuildID:    resp.BuildID,
		TemplateID: resp.TemplateID,
		Status:     resp.Status,
		Logs:       resp.Logs,
		LogEntries: resp.LogEntries,
		Reason:     resp.Reason,
	}, nil
}

func (c *Client) WaitForBuild(ctx context.Context, templateID, buildID string, opts ...BuildOption) (*BuildStatusResponse, error) {
	cfg := applyBuildOpts(opts)

	logsOffset := 0
	pollInterval := 200 * time.Millisecond
	const maxPollInterval = 2 * time.Second

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		status, err := c.GetBuildStatus(ctx, templateID, buildID, logsOffset)
		if err != nil {
			return nil, err
		}

		if cfg.onBuildLogs != nil {
			for _, entry := range status.LogEntries {
				cfg.onBuildLogs(entry)
			}
		}
		logsOffset += len(status.LogEntries)

		switch status.Status {
		case BuildStatusReady:
			return status, nil
		case BuildStatusError:
			msg := "template build failed"
			if status.Reason != nil {
				msg = status.Reason.Message
			}
			return status, &BuildError{
				SandboxError: SandboxError{Message: msg},
				BuildID:      buildID,
				TemplateID:   templateID,
			}
		case BuildStatusBuilding, BuildStatusWaiting:
		default:
			return status, fmt.Errorf("unknown build status: %s", status.Status)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}

		if pollInterval < maxPollInterval {
			pollInterval *= 2
			if pollInterval > maxPollInterval {
				pollInterval = maxPollInterval
			}
		}
	}
}
