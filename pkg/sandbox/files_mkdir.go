package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/filesystem"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// MakeDir creates a directory, along with any missing parents. It reports false
// when the directory already existed.
func (f *Filesystem) MakeDir(ctx context.Context, path string, opts FileOptions) (bool, error) {
	req := &filesystem.MakeDirRequest{Path: path}

	if _, err := f.sbx.conn.filesystem.MakeDir(ctx, requestFor(req, f.sbx, opts.User)); err != nil {
		mapped := errdefsFromConnect(err)
		if errdefs.IsConflict(mapped) {
			return false, nil
		}
		return false, mapped
	}
	return true, nil
}
