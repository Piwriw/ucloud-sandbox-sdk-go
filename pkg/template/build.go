package template

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// Polling bounds for WaitForBuild. The interval starts short so a cached build
// is noticed almost at once, then backs off so a long build does not hammer the
// control plane.
const (
	initialPollInterval = 200 * time.Millisecond
	maxPollInterval     = 2 * time.Second
)

// BuildOptions are the optional arguments to Build.
type BuildOptions struct {
	// Tags to assign to the resulting build.
	Tags []string

	// CPUCount for sandboxes started from the template. Zero uses the team's
	// default.
	CPUCount int

	// MemoryMB for sandboxes started from the template. Zero uses the team's
	// default.
	MemoryMB int

	// MinFreeDiskMB is the free space to leave after the build steps have run.
	// Zero uses the team's default; a pointer to 0 asks for no growth.
	MinFreeDiskMB *int

	// SkipCache rebuilds every step, ignoring cached results.
	SkipCache bool

	// Publish makes the template public once the build succeeds.
	Publish bool

	// OnLogs receives each log entry as the build produces it. DefaultLogger
	// prints them to stderr.
	OnLogs func(LogEntry)

	// Registry overrides the credentials for pulling the base image. Usually
	// set on the builder instead, with Builder.FromImageWithAuth.
	Registry *Registry
}

// Build runs the whole build sequence and waits for it to finish: register the
// template, upload any files its COPY steps need, start the build, then poll
// until it succeeds or fails.
//
// name may carry a tag, as in "my-template:v1".
//
// The returned BuildInfo is filled in as soon as the build is registered, so it
// is returned alongside an error too — a failed build still has an ID worth
// reporting, and its logs can be fetched with BuildLogs.
func (s *Service) Build(ctx context.Context, b *Builder, name string, opts BuildOptions) (*BuildInfo, error) {
	if b.err != nil {
		return nil, &errdefs.TemplateError{SandboxError: errdefs.SandboxError{
			Message: b.err.Error(), Cause: b.err,
		}}
	}
	if opts.Registry != nil {
		b.registry = opts.Registry
	}

	// Hashing the COPY bundles first, because the hashes are part of the build
	// request and decide which bundles have to be uploaded.
	steps, err := b.prepareSteps()
	if err != nil {
		return nil, &errdefs.TemplateError{SandboxError: errdefs.SandboxError{
			Message: err.Error(), Cause: err,
		}}
	}

	emit(opts.OnLogs, "Requesting build for template: %s", name)

	info, err := s.CreateV3(ctx, name, CreateV3Options{
		Tags:          opts.Tags,
		CPUCount:      opts.CPUCount,
		MemoryMB:      opts.MemoryMB,
		MinFreeDiskMB: opts.MinFreeDiskMB,
	})
	if err != nil {
		return nil, err
	}

	emit(opts.OnLogs, "Template created with ID %s, build %s", info.TemplateID, info.BuildID)

	if err := s.uploadFiles(ctx, info.TemplateID, b, steps); err != nil {
		return info, err
	}

	emit(opts.OnLogs, "Starting build...")

	if err := s.StartBuildV2(ctx, info.TemplateID, info.BuildID, b, StartBuildV2Options{
		SkipCache: opts.SkipCache,
		Steps:     steps,
	}); err != nil {
		return info, err
	}

	if _, err := s.WaitForBuild(ctx, info.TemplateID, info.BuildID, opts.OnLogs); err != nil {
		return info, err
	}

	if opts.Publish {
		emit(opts.OnLogs, "Publishing template...")
		if err := s.Publish(ctx, info.TemplateID); err != nil {
			return info, err
		}
		emit(opts.OnLogs, "Template published")
	}

	return info, nil
}

// WaitForBuild polls until a build succeeds or fails, forwarding each new log
// entry to onLogs. Pass nil for onLogs to wait silently.
//
// A build that ends in error returns the final status together with an
// *errdefs.BuildError, so the caller can read the reason off either.
func (s *Service) WaitForBuild(ctx context.Context, templateID, buildID string, onLogs func(LogEntry)) (*Status, error) {
	logsOffset := 0
	interval := initialPollInterval

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		status, err := s.BuildStatus(ctx, templateID, buildID, BuildStatusOptions{LogsOffset: logsOffset})
		if err != nil {
			return nil, err
		}

		if onLogs != nil {
			for _, entry := range status.LogEntries {
				onLogs(entry)
			}
		}
		logsOffset += len(status.LogEntries)

		switch status.Status {
		case BuildStatusReady:
			return status, nil

		case BuildStatusError:
			message := "template build failed"
			if status.Reason != nil && status.Reason.Message != "" {
				message = status.Reason.Message
			}
			return status, &errdefs.BuildError{
				SandboxError: errdefs.SandboxError{Message: message},
				BuildID:      buildID,
				TemplateID:   templateID,
			}

		case BuildStatusBuilding, BuildStatusWaiting:
			// Still going.

		default:
			return status, &errdefs.BuildError{
				SandboxError: errdefs.SandboxError{
					Message: fmt.Sprintf("unknown build status %q", status.Status),
				},
				BuildID:    buildID,
				TemplateID: templateID,
			}
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		interval = min(interval*2, maxPollInterval)
	}
}

// emit sends a message from the SDK itself to the log callback, so that the
// build's own output and the steps around it read as one stream.
func emit(onLogs func(LogEntry), format string, args ...any) {
	if onLogs == nil {
		return
	}
	onLogs(LogEntry{
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   fmt.Sprintf(format, args...),
	})
}

// DefaultLogger returns a log callback that prints entries of level info and
// above to stderr.
func DefaultLogger() func(LogEntry) {
	return DefaultLoggerWithLevel(LogLevelInfo)
}

// DefaultLoggerWithLevel returns a log callback that prints entries at or above
// minLevel to stderr, prefixed with the elapsed time.
func DefaultLoggerWithLevel(minLevel LogLevel) func(LogEntry) {
	start := time.Now()
	minOrd := levelOrder(minLevel)

	return func(entry LogEntry) {
		if levelOrder(entry.Level) < minOrd {
			return
		}
		// A build log entry can arrive without a timestamp; showing the
		// current time is more use than showing the zero time.
		stamp := entry.Timestamp
		if stamp.IsZero() {
			stamp = time.Now()
		}
		fmt.Fprintf(os.Stderr, "%5.1fs | %s %-5s %s\n",
			time.Since(start).Seconds(),
			stamp.Format("15:04:05"),
			strings.ToUpper(string(entry.Level)),
			entry.Message)
	}
}

// levelOrder ranks a severity, treating anything unrecognised as info.
func levelOrder(level LogLevel) int {
	switch level {
	case LogLevelDebug:
		return 0
	case LogLevelWarn:
		return 2
	case LogLevelError:
		return 3
	default:
		return 1
	}
}
