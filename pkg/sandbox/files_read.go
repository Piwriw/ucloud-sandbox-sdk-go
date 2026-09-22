package sandbox

import (
	"context"
	"io"
	"net/http"

	envdapi "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// Read returns a file's contents as a string.
func (f *Filesystem) Read(ctx context.Context, path string, opts FileOptions) (string, error) {
	data, err := f.ReadBytes(ctx, path, opts)
	return string(data), err
}

// ReadBytes returns a file's contents.
func (f *Filesystem) ReadBytes(ctx context.Context, path string, opts FileOptions) ([]byte, error) {
	reader, err := f.ReadStream(ctx, path, opts)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// ReadStream streams a file's contents. The caller must close the reader.
//
// Prefer this over Read for anything large enough that holding it in memory
// matters.
func (f *Filesystem) ReadStream(ctx context.Context, path string, opts FileOptions) (io.ReadCloser, error) {
	params := &envdapi.GetFilesParams{Path: &path}
	if user := f.sbx.resolveUser(opts.User); user != "" {
		params.Username = &user
	}

	resp, err := f.sbx.conn.files.GetFiles(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := checkEnvdResponse(resp); err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// checkEnvdResponse turns a non-2xx envd response into an error, closing the
// body on the way.
func checkEnvdResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	return errdefs.FromHTTP(resp.StatusCode, string(body))
}
