package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/filesystem"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// Watcher is a polled directory watch, the non-streaming alternative to Watch.
//
// Watch holds a long-lived stream, which is the better fit when one is
// available. A Watcher instead parks state inside envd and hands out events on
// demand, which survives a client that cannot keep a connection open: a
// serverless function, or anything behind a proxy that cuts idle streams.
type Watcher struct {
	// ID identifies the watcher to envd.
	ID string

	fs   *Filesystem
	user string
}

// CreateWatcher starts a polled watch on a directory.
//
// The watcher lives inside the sandbox until RemoveWatcher, so a caller that
// abandons one leaves it running.
func (f *Filesystem) CreateWatcher(ctx context.Context, path string, opts WatchOptions) (*Watcher, error) {
	if opts.Recursive && !f.sbx.envdVersion.supports(envdVersionRecursiveWatch) {
		return nil, &errdefs.SandboxError{
			Message: "recursive watching needs envd 0.1.4 or newer; rebuild the template",
		}
	}

	req := &filesystem.CreateWatcherRequest{
		Path:               path,
		Recursive:          opts.Recursive,
		IncludeEntry:       opts.IncludeEntry,
		AllowNetworkMounts: opts.AllowNetworkMounts,
	}

	resp, err := f.sbx.conn.filesystem.CreateWatcher(ctx, requestFor(req, f.sbx, opts.User))
	if err != nil {
		return nil, errdefsFromConnect(err)
	}
	return &Watcher{ID: resp.Msg.GetWatcherId(), fs: f, user: opts.User}, nil
}

// Events returns the changes seen since the last call. The result is never nil,
// and is empty when nothing has happened.
func (w *Watcher) Events(ctx context.Context) ([]FilesystemEvent, error) {
	req := &filesystem.GetWatcherEventsRequest{WatcherId: w.ID}

	resp, err := w.fs.sbx.conn.filesystem.GetWatcherEvents(ctx, requestFor(req, w.fs.sbx, w.user))
	if err != nil {
		return nil, errdefsFromConnect(err)
	}

	reported := resp.Msg.GetEvents()
	events := make([]FilesystemEvent, 0, len(reported))
	for _, event := range reported {
		events = append(events, eventFrom(event))
	}
	return events, nil
}

// Remove stops the watcher and frees it inside the sandbox.
func (w *Watcher) Remove(ctx context.Context) error {
	req := &filesystem.RemoveWatcherRequest{WatcherId: w.ID}

	if _, err := w.fs.sbx.conn.filesystem.RemoveWatcher(ctx, requestFor(req, w.fs.sbx, w.user)); err != nil {
		return errdefsFromConnect(err)
	}
	return nil
}
