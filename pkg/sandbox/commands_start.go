package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process"
)

// Start runs a command and returns without waiting for it. Use the handle to
// read output as it arrives, feed stdin, wait, or kill.
//
// The command runs through a login shell, so pipes, redirection and globbing
// work as typed.
func (c *Commands) Start(ctx context.Context, cmd string, opts CommandOptions) (*CommandHandle, error) {
	req := &process.StartRequest{Process: c.processConfig(cmd, opts)}

	// envd defaults stdin to open for backwards compatibility. Newer agents
	// let the client choose, and leaving it closed is the better default: a
	// command reading from a stdin nobody writes to would hang.
	if c.sbx.envdVersion.supports(envdVersionStdin) {
		stdin := opts.Stdin
		req.Stdin = &stdin
	}

	stream, err := c.sbx.conn.process.Start(ctx, requestFor(req, c.sbx, opts.User))
	if err != nil {
		return nil, errdefsFromConnect(err)
	}

	handle := &CommandHandle{cmds: c, done: make(chan struct{})}
	go handle.consume(startStream{stream}, opts)

	return handle, nil
}
