package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/filesystem"
)

// Remove deletes a file, or a directory and everything under it.
func (f *Filesystem) Remove(ctx context.Context, path string, opts FileOptions) error {
	req := &filesystem.RemoveRequest{Path: path}

	if _, err := f.sbx.conn.filesystem.Remove(ctx, requestFor(req, f.sbx, opts.User)); err != nil {
		return errdefsFromConnect(err)
	}
	return nil
}
