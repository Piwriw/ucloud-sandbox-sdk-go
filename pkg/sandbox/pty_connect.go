package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process"
)

// Connect attaches to a pseudo-terminal already open in the sandbox, so it can
// be driven from a different client than the one that created it.
func (p *Pty) Connect(ctx context.Context, pid int, opts CommandOptions) (*PtyHandle, error) {
	req := &process.ConnectRequest{Process: selectorForPID(pid)}

	stream, err := p.sbx.conn.process.Connect(ctx, requestFor(req, p.sbx, opts.User))
	if err != nil {
		return nil, errdefsFromConnect(err)
	}

	handle := newPtyHandle(p, pid)
	go handle.consume(connectStream{stream})

	return handle, nil
}
