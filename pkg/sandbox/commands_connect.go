package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process"
)

// Connect attaches to a process already running in the sandbox, so its output
// can be followed from a different client than the one that started it.
//
// Output produced before connecting is not replayed.
func (c *Commands) Connect(ctx context.Context, pid int, opts CommandOptions) (*CommandHandle, error) {
	req := &process.ConnectRequest{Process: selectorForPID(pid)}

	stream, err := c.sbx.conn.process.Connect(ctx, requestFor(req, c.sbx, opts.User))
	if err != nil {
		return nil, errdefsFromConnect(err)
	}

	handle := &CommandHandle{PID: pid, cmds: c, done: make(chan struct{})}
	go handle.consume(connectStream{stream}, opts)

	return handle, nil
}
