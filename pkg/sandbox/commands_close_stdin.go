package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process"
)

// CloseStdin closes a process's stdin, signalling EOF to a command that reads
// until the input runs out.
//
// This has no effect on a PTY; send Ctrl-D (0x04) there instead.
func (c *Commands) CloseStdin(ctx context.Context, pid int) error {
	req := &process.CloseStdinRequest{Process: selectorForPID(pid)}

	if _, err := c.sbx.conn.process.CloseStdin(ctx, requestFor(req, c.sbx, "")); err != nil {
		return errdefsFromConnect(err)
	}
	return nil
}
