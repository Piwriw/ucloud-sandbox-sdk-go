package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// Pty opens pseudo-terminals inside a sandbox. Reach it through Sandbox.Pty.
//
// A PTY differs from a command in that its output is one interleaved byte
// stream rather than separate stdout and stderr, and it carries terminal
// control sequences. It is what an interactive shell needs.
type Pty struct {
	sbx *Sandbox
}

// PtyHandle is an open pseudo-terminal.
type PtyHandle struct {
	// PID is the process ID inside the sandbox.
	PID int

	pty    *Pty
	output chan []byte
	done   chan struct{}
	err    error
	result *CommandResult
}

// Output delivers the terminal's bytes as they arrive. It is closed when the
// terminal ends.
func (h *PtyHandle) Output() <-chan []byte { return h.output }

// Wait blocks until the terminal ends and reports how it exited.
func (h *PtyHandle) Wait(ctx context.Context) (*CommandResult, error) {
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

// Kill terminates the terminal's process.
func (h *PtyHandle) Kill(ctx context.Context) (bool, error) {
	return h.pty.Kill(ctx, h.PID)
}

// SendStdin writes bytes to the terminal. Send 0x04 for Ctrl-D, which is how a
// PTY signals EOF.
func (h *PtyHandle) SendStdin(ctx context.Context, data []byte) error {
	return h.pty.SendStdin(ctx, h.PID, data)
}

// Resize tells the terminal its new dimensions, so full-screen programs redraw
// correctly.
func (h *PtyHandle) Resize(ctx context.Context, size PtySize) error {
	return h.pty.Resize(ctx, h.PID, size)
}

// ptySize converts a size into the proto message.
func ptySize(size PtySize) *process.PTY {
	return &process.PTY{
		Size: &process.PTY_Size{
			Cols: uint32(size.Cols),
			Rows: uint32(size.Rows),
		},
	}
}

// consume drains a PTY stream into the handle.
func (h *PtyHandle) consume(stream processStream) {
	defer close(h.done)
	defer close(h.output)
	defer stream.Close()

	for stream.Receive() {
		switch event := stream.event().GetEvent().(type) {
		case *process.ProcessEvent_Start:
			h.PID = int(event.Start.GetPid())

		case *process.ProcessEvent_Data:
			if data, ok := event.Data.GetOutput().(*process.ProcessEvent_DataEvent_Pty); ok {
				h.output <- data.Pty
			}

		case *process.ProcessEvent_End:
			h.result = errFromEnd(event.End, "", "")

		case *process.ProcessEvent_Keepalive:
		}
	}

	if err := stream.Err(); err != nil {
		h.err = errdefsFromConnect(err)
		return
	}
	if h.result == nil {
		h.err = errStreamEndedEarly()
	}
}

// Kill terminates a terminal's process. One already gone is reported as false.
func (p *Pty) Kill(ctx context.Context, pid int) (bool, error) {
	req := &process.SendSignalRequest{
		Process: selectorForPID(pid),
		Signal:  process.Signal_SIGNAL_SIGKILL,
	}

	if _, err := p.sbx.conn.process.SendSignal(ctx, requestFor(req, p.sbx, "")); err != nil {
		mapped := errdefsFromConnect(err)
		if errdefs.IsNotFound(mapped) {
			return false, nil
		}
		return false, mapped
	}
	return true, nil
}

// SendStdin writes bytes to a terminal.
func (p *Pty) SendStdin(ctx context.Context, pid int, data []byte) error {
	req := &process.SendInputRequest{
		Process: selectorForPID(pid),
		Input: &process.ProcessInput{
			Input: &process.ProcessInput_Pty{Pty: data},
		},
	}

	if _, err := p.sbx.conn.process.SendInput(ctx, requestFor(req, p.sbx, "")); err != nil {
		return errdefsFromConnect(err)
	}
	return nil
}

// Resize tells a terminal its new dimensions.
func (p *Pty) Resize(ctx context.Context, pid int, size PtySize) error {
	req := &process.UpdateRequest{
		Process: selectorForPID(pid),
		Pty:     ptySize(size),
	}

	if _, err := p.sbx.conn.process.Update(ctx, requestFor(req, p.sbx, "")); err != nil {
		return errdefsFromConnect(err)
	}
	return nil
}

// ptyOutputBuffer is how many chunks a PTY buffers before a slow reader starts
// holding up the stream.
const ptyOutputBuffer = 64

// newPtyHandle builds a handle with its channels ready.
func newPtyHandle(p *Pty, pid int) *PtyHandle {
	return &PtyHandle{
		PID:    pid,
		pty:    p,
		output: make(chan []byte, ptyOutputBuffer),
		done:   make(chan struct{}),
	}
}
