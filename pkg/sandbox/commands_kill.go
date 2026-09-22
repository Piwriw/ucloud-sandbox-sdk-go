package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// Kill sends SIGKILL to a process. A process that is already gone is reported
// as false rather than as an error.
func (c *Commands) Kill(ctx context.Context, pid int) (bool, error) {
	req := &process.SendSignalRequest{
		Process: selectorForPID(pid),
		Signal:  process.Signal_SIGNAL_SIGKILL,
	}

	if _, err := c.sbx.conn.process.SendSignal(ctx, requestFor(req, c.sbx, "")); err != nil {
		mapped := errdefsFromConnect(err)
		if errdefs.IsNotFound(mapped) {
			return false, nil
		}
		return false, mapped
	}
	return true, nil
}
