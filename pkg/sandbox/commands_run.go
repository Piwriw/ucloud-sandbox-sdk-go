package sandbox

import (
	"context"
	"time"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// Run executes a command and waits for it to finish.
//
// A command that exits non-zero is returned as an *errdefs.CommandExitError
// carrying the captured output, so a failure is not silently mistaken for
// success. The result is returned alongside it either way.
func (c *Commands) Run(ctx context.Context, cmd string, opts CommandOptions) (*CommandResult, error) {
	ctx, cancel := commandContext(ctx, opts)
	defer cancel()

	handle, err := c.Start(ctx, cmd, opts)
	if err != nil {
		return nil, err
	}

	result, err := handle.Wait(ctx)
	if err != nil {
		return result, err
	}
	if result.ExitCode != 0 {
		return result, &errdefs.CommandExitError{
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: result.ExitCode,
			Message:  orDefault(result.Error, result.Stderr),
		}
	}
	return result, nil
}

// commandContext applies the command timeout.
//
// Zero means the default; a negative value means no timeout at all, for a
// command expected to outlive it.
func commandContext(ctx context.Context, opts CommandOptions) (context.Context, context.CancelFunc) {
	switch {
	case opts.TimeoutSeconds < 0:
		return ctx, func() {}
	case opts.TimeoutSeconds == 0:
		return context.WithTimeout(ctx, DefaultCommandTimeoutSeconds*time.Second)
	default:
		return context.WithTimeout(ctx, time.Duration(opts.TimeoutSeconds)*time.Second)
	}
}

// Wait blocks until the command finishes and returns what it produced.
func (h *CommandHandle) Wait(ctx context.Context) (*CommandResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.done:
	}

	if h.err != nil {
		return nil, h.err
	}
	return h.result, nil
}

// Kill terminates the command.
func (h *CommandHandle) Kill(ctx context.Context) (bool, error) {
	return h.cmds.Kill(ctx, h.PID)
}

// SendStdin writes to the command's stdin. The command must have been started
// with CommandOptions.Stdin set.
func (h *CommandHandle) SendStdin(ctx context.Context, data string) error {
	return h.cmds.SendStdin(ctx, h.PID, data)
}

// CloseStdin closes the command's stdin, signalling EOF.
func (h *CommandHandle) CloseStdin(ctx context.Context) error {
	return h.cmds.CloseStdin(ctx, h.PID)
}
