package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/filesystem"
)

// List returns the entries under a directory. The result is never nil.
//
// Set FileOptions.Depth to descend further than the immediate children.
func (f *Filesystem) List(ctx context.Context, path string, opts FileOptions) ([]EntryInfo, error) {
	req := &filesystem.ListDirRequest{Path: path, Depth: opts.Depth}

	resp, err := f.sbx.conn.filesystem.ListDir(ctx, requestFor(req, f.sbx, opts.User))
	if err != nil {
		return nil, errdefsFromConnect(err)
	}

	listed := resp.Msg.GetEntries()
	entries := make([]EntryInfo, 0, len(listed))
	for _, entry := range listed {
		entries = append(entries, *entryFrom(entry))
	}
	return entries, nil
}
