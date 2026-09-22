package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process"
)

// Create opens a pseudo-terminal running a login shell, sized as given.
//
// The caller must drain PtyHandle.Output, or the terminal stalls once its
// buffer fills.
func (p *Pty) Create(ctx context.Context, size PtySize, opts CommandOptions) (*PtyHandle, error) {
	config := &process.ProcessConfig{
		Cmd:  "/bin/bash",
		Args: []string{"-i", "-l"},
		Envs: map[string]string{"TERM": "xterm-256color"},
	}
	for key, value := range opts.EnvVars {
		config.Envs[key] = value
	}
	if opts.Cwd != "" {
		config.Cwd = &opts.Cwd
	}

	stdin := true
	req := &process.StartRequest{
		Process: config,
		Pty:     ptySize(size),
		Stdin:   &stdin,
	}

	stream, err := p.sbx.conn.process.Start(ctx, requestFor(req, p.sbx, opts.User))
	if err != nil {
		return nil, errdefsFromConnect(err)
	}

	handle := newPtyHandle(p, 0)
	go handle.consume(startStream{stream})

	return handle, nil
}
