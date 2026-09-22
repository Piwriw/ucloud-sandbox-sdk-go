package sandbox

import (
	"connectrpc.com/connect"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process"
)

// processStream is the half of a connect server stream that Start and Connect
// share: both deliver process.ProcessEvent, just wrapped in different response
// types.
type processStream interface {
	Receive() bool
	Err() error
	Close() error
	event() *process.ProcessEvent
}

type startStream struct {
	*connect.ServerStreamForClient[process.StartResponse]
}

func (s startStream) event() *process.ProcessEvent { return s.Msg().GetEvent() }

type connectStream struct {
	*connect.ServerStreamForClient[process.ConnectResponse]
}

func (s connectStream) event() *process.ProcessEvent { return s.Msg().GetEvent() }

// consume drains a process stream into the handle, then closes it.
//
// It runs in its own goroutine for Start and Connect, and inline for Run. The
// handle's done channel is closed exactly once, when the stream ends.
func (h *CommandHandle) consume(stream processStream, opts CommandOptions) {
	defer close(h.done)
	defer stream.Close()

	for stream.Receive() {
		switch event := stream.event().GetEvent().(type) {
		case *process.ProcessEvent_Start:
			h.PID = int(event.Start.GetPid())

		case *process.ProcessEvent_Data:
			switch data := event.Data.GetOutput().(type) {
			case *process.ProcessEvent_DataEvent_Stdout:
				h.appendOutput(&h.stdout, data.Stdout, opts.OnStdout)
			case *process.ProcessEvent_DataEvent_Stderr:
				h.appendOutput(&h.stderr, data.Stderr, opts.OnStderr)
			}

		case *process.ProcessEvent_End:
			h.result = errFromEnd(event.End, h.Stdout(), h.Stderr())

		case *process.ProcessEvent_Keepalive:
			// Only there to keep the connection from going idle.
		}
	}

	if err := stream.Err(); err != nil {
		h.err = errdefsFromConnect(err)
		return
	}
	if h.result == nil {
		// The stream ended without an end event, so the command's fate is
		// unknown. Reporting a zero exit code here would claim success.
		h.err = errStreamEndedEarly()
	}
}
