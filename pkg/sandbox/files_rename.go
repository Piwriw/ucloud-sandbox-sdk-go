package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/filesystem"
)

// Rename moves an entry and returns its metadata at the new path.
func (f *Filesystem) Rename(ctx context.Context, oldPath, newPath string, opts FileOptions) (*EntryInfo, error) {
	req := &filesystem.MoveRequest{Source: oldPath, Destination: newPath}

	resp, err := f.sbx.conn.filesystem.Move(ctx, requestFor(req, f.sbx, opts.User))
	if err != nil {
		return nil, errdefsFromConnect(err)
	}
	return entryFrom(resp.Msg.GetEntry()), nil
}
