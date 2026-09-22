package sandbox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process/processconnect"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// fakeProcess is a connect handler standing in for envd's process service. The
// zero value answers every method with "unimplemented"; a test fills in the one
// it exercises.
type fakeProcess struct {
	processconnect.UnimplementedProcessHandler

	start       func(context.Context, *connect.Request[process.StartRequest], *connect.ServerStream[process.StartResponse]) error
	connectProc func(context.Context, *connect.Request[process.ConnectRequest], *connect.ServerStream[process.ConnectResponse]) error
	list        func(context.Context, *connect.Request[process.ListRequest]) (*connect.Response[process.ListResponse], error)
	sendInput   func(context.Context, *connect.Request[process.SendInputRequest]) (*connect.Response[process.SendInputResponse], error)
	sendSignal  func(context.Context, *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error)
	closeStdin  func(context.Context, *connect.Request[process.CloseStdinRequest]) (*connect.Response[process.CloseStdinResponse], error)
}

func (f *fakeProcess) Start(ctx context.Context, req *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
	return f.start(ctx, req, stream)
}

func (f *fakeProcess) Connect(ctx context.Context, req *connect.Request[process.ConnectRequest], stream *connect.ServerStream[process.ConnectResponse]) error {
	return f.connectProc(ctx, req, stream)
}

func (f *fakeProcess) List(ctx context.Context, req *connect.Request[process.ListRequest]) (*connect.Response[process.ListResponse], error) {
	return f.list(ctx, req)
}

func (f *fakeProcess) SendInput(ctx context.Context, req *connect.Request[process.SendInputRequest]) (*connect.Response[process.SendInputResponse], error) {
	return f.sendInput(ctx, req)
}

func (f *fakeProcess) SendSignal(ctx context.Context, req *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error) {
	return f.sendSignal(ctx, req)
}

func (f *fakeProcess) CloseStdin(ctx context.Context, req *connect.Request[process.CloseStdinRequest]) (*connect.Response[process.CloseStdinResponse], error) {
	return f.closeStdin(ctx, req)
}

// newEnvdSandbox stands up a connect server for handler and returns a Sandbox
// pointed at it, as if the control plane had just created one.
func newEnvdSandbox(t *testing.T, handler processconnect.ProcessHandler, envdVersion string) *Sandbox {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle(processconnect.NewProcessHandler(handler))

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	tc, err := transport.New(transport.Config{
		APIKey:     "test-key",
		APIURL:     srv.URL,
		SandboxURL: srv.URL,
	})
	require.NoError(t, err)

	svc := NewService(tc)
	sbx, err := svc.newSandbox("sbx-1", apiSandbox(envdVersion))
	require.NoError(t, err)
	return sbx
}

// startEvent, dataEvent and endEvent build the three process events a stream
// carries.
func startEvent(pid int) *process.ProcessEvent {
	return &process.ProcessEvent{Event: &process.ProcessEvent_Start{
		Start: &process.ProcessEvent_StartEvent{Pid: uint32(pid)},
	}}
}

func stdoutEvent(text string) *process.ProcessEvent {
	return &process.ProcessEvent{Event: &process.ProcessEvent_Data{
		Data: &process.ProcessEvent_DataEvent{
			Output: &process.ProcessEvent_DataEvent_Stdout{Stdout: []byte(text)},
		},
	}}
}

func stderrEvent(text string) *process.ProcessEvent {
	return &process.ProcessEvent{Event: &process.ProcessEvent_Data{
		Data: &process.ProcessEvent_DataEvent{
			Output: &process.ProcessEvent_DataEvent_Stderr{Stderr: []byte(text)},
		},
	}}
}

func endEvent(exitCode int32) *process.ProcessEvent {
	return &process.ProcessEvent{Event: &process.ProcessEvent_End{
		End: &process.ProcessEvent_EndEvent{ExitCode: exitCode, Exited: true},
	}}
}

func TestRunCollectsStreamedOutput(t *testing.T) {
	var gotCmd []string

	sbx := newEnvdSandbox(t, &fakeProcess{
		start: func(_ context.Context, req *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			gotCmd = req.Msg.GetProcess().GetArgs()

			for _, event := range []*process.ProcessEvent{
				startEvent(42),
				stdoutEvent("hello "),
				stdoutEvent("world\n"),
				stderrEvent("a warning\n"),
				endEvent(0),
			} {
				if err := stream.Send(&process.StartResponse{Event: event}); err != nil {
					return err
				}
			}
			return nil
		},
	}, "0.4.0")

	result, err := sbx.Commands.Run(context.Background(), "echo hello world", CommandOptions{})
	require.NoError(t, err)

	// Chunks arriving separately must be joined, not overwritten.
	assert.Equal(t, "hello world\n", result.Stdout)
	assert.Equal(t, "a warning\n", result.Stderr)
	assert.Zero(t, result.ExitCode)

	// The command goes through a login shell, which is what makes pipes and
	// globbing work.
	assert.Equal(t, []string{"-l", "-c", "echo hello world"}, gotCmd)
}

func TestRunReportsNonZeroExit(t *testing.T) {
	sbx := newEnvdSandbox(t, &fakeProcess{
		start: func(_ context.Context, _ *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			_ = stream.Send(&process.StartResponse{Event: startEvent(1)})
			_ = stream.Send(&process.StartResponse{Event: stderrEvent("no such file")})
			return stream.Send(&process.StartResponse{Event: endEvent(127)})
		},
	}, "0.4.0")

	result, err := sbx.Commands.Run(context.Background(), "nope", CommandOptions{})
	require.Error(t, err)

	// The result comes back alongside the error, since the output is usually
	// the only explanation of what went wrong.
	require.NotNil(t, result)
	assert.Equal(t, 127, result.ExitCode)

	var exitErr *errdefs.CommandExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 127, exitErr.ExitCode)
	assert.Contains(t, exitErr.Stderr, "no such file")
}

func TestRunCallbacksSeeOutputAsItArrives(t *testing.T) {
	sbx := newEnvdSandbox(t, &fakeProcess{
		start: func(_ context.Context, _ *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			_ = stream.Send(&process.StartResponse{Event: startEvent(1)})
			_ = stream.Send(&process.StartResponse{Event: stdoutEvent("one\n")})
			_ = stream.Send(&process.StartResponse{Event: stdoutEvent("two\n")})
			_ = stream.Send(&process.StartResponse{Event: stderrEvent("err\n")})
			return stream.Send(&process.StartResponse{Event: endEvent(0)})
		},
	}, "0.4.0")

	var stdout, stderr []string
	_, err := sbx.Commands.Run(context.Background(), "x", CommandOptions{
		OnStdout: func(s string) { stdout = append(stdout, s) },
		OnStderr: func(s string) { stderr = append(stderr, s) },
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"one\n", "two\n"}, stdout)
	assert.Equal(t, []string{"err\n"}, stderr)
}

func TestStreamEndingWithoutAnExitStatusIsAnError(t *testing.T) {
	sbx := newEnvdSandbox(t, &fakeProcess{
		start: func(_ context.Context, _ *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			// Start and some output, then the stream simply ends.
			_ = stream.Send(&process.StartResponse{Event: startEvent(1)})
			return stream.Send(&process.StartResponse{Event: stdoutEvent("partial")})
		},
	}, "0.4.0")

	_, err := sbx.Commands.Run(context.Background(), "x", CommandOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "without reporting an exit status",
		"a truncated stream must not be reported as a successful exit")
}

func TestStartStdinFlagFollowsEnvdVersion(t *testing.T) {
	tests := map[string]struct {
		envdVersion string
		stdin       bool
		wantSet     bool
	}{
		"new envd, stdin off": {"0.4.0", false, true},
		"new envd, stdin on":  {"0.4.0", true, true},
		// envd before 0.3.0 has no stdin flag and always leaves it open, so
		// sending the field would be meaningless.
		"old envd": {"0.2.0", false, false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var got *process.StartRequest

			sbx := newEnvdSandbox(t, &fakeProcess{
				start: func(_ context.Context, req *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
					got = req.Msg
					_ = stream.Send(&process.StartResponse{Event: startEvent(1)})
					return stream.Send(&process.StartResponse{Event: endEvent(0)})
				},
			}, tt.envdVersion)

			_, err := sbx.Commands.Run(context.Background(), "x", CommandOptions{Stdin: tt.stdin})
			require.NoError(t, err)

			require.NotNil(t, got)
			if !tt.wantSet {
				assert.Nil(t, got.Stdin)
				return
			}
			require.NotNil(t, got.Stdin)
			assert.Equal(t, tt.stdin, *got.Stdin)
		})
	}
}

func TestStartReturnsBeforeTheCommandFinishes(t *testing.T) {
	release := make(chan struct{})

	sbx := newEnvdSandbox(t, &fakeProcess{
		start: func(_ context.Context, _ *connect.Request[process.StartRequest], stream *connect.ServerStream[process.StartResponse]) error {
			_ = stream.Send(&process.StartResponse{Event: startEvent(7)})
			<-release
			_ = stream.Send(&process.StartResponse{Event: stdoutEvent("done\n")})
			return stream.Send(&process.StartResponse{Event: endEvent(0)})
		},
	}, "0.4.0")

	handle, err := sbx.Commands.Start(context.Background(), "sleep", CommandOptions{})
	require.NoError(t, err)

	close(release)

	result, err := handle.Wait(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "done\n", result.Stdout)
	assert.Equal(t, 7, handle.PID)
}

func TestList(t *testing.T) {
	cwd := "/app"
	sbx := newEnvdSandbox(t, &fakeProcess{
		list: func(context.Context, *connect.Request[process.ListRequest]) (*connect.Response[process.ListResponse], error) {
			return connect.NewResponse(&process.ListResponse{
				Processes: []*process.ProcessInfo{{
					Pid: 12,
					Config: &process.ProcessConfig{
						Cmd:  "/bin/bash",
						Args: []string{"-c", "sleep 100"},
						Envs: map[string]string{"A": "1"},
						Cwd:  &cwd,
					},
				}},
			}), nil
		},
	}, "0.4.0")

	processes, err := sbx.Commands.List(context.Background())
	require.NoError(t, err)

	require.Len(t, processes, 1)
	assert.Equal(t, ProcessInfo{
		PID:  12,
		Cmd:  "/bin/bash",
		Args: []string{"-c", "sleep 100"},
		Envs: map[string]string{"A": "1"},
		Cwd:  "/app",
	}, processes[0])
}

func TestKill(t *testing.T) {
	t.Run("running process", func(t *testing.T) {
		var got *process.SendSignalRequest
		sbx := newEnvdSandbox(t, &fakeProcess{
			sendSignal: func(_ context.Context, req *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error) {
				got = req.Msg
				return connect.NewResponse(&process.SendSignalResponse{}), nil
			},
		}, "0.4.0")

		ok, err := sbx.Commands.Kill(context.Background(), 42)
		require.NoError(t, err)
		assert.True(t, ok)

		assert.Equal(t, process.Signal_SIGNAL_SIGKILL, got.GetSignal())
		assert.EqualValues(t, 42, got.GetProcess().GetPid())
	})

	t.Run("missing process is false, not an error", func(t *testing.T) {
		sbx := newEnvdSandbox(t, &fakeProcess{
			sendSignal: func(context.Context, *connect.Request[process.SendSignalRequest]) (*connect.Response[process.SendSignalResponse], error) {
				return nil, connect.NewError(connect.CodeNotFound, assertNotFound)
			},
		}, "0.4.0")

		ok, err := sbx.Commands.Kill(context.Background(), 42)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestSendAndCloseStdin(t *testing.T) {
	var input *process.SendInputRequest
	var closed *process.CloseStdinRequest

	sbx := newEnvdSandbox(t, &fakeProcess{
		sendInput: func(_ context.Context, req *connect.Request[process.SendInputRequest]) (*connect.Response[process.SendInputResponse], error) {
			input = req.Msg
			return connect.NewResponse(&process.SendInputResponse{}), nil
		},
		closeStdin: func(_ context.Context, req *connect.Request[process.CloseStdinRequest]) (*connect.Response[process.CloseStdinResponse], error) {
			closed = req.Msg
			return connect.NewResponse(&process.CloseStdinResponse{}), nil
		},
	}, "0.4.0")

	require.NoError(t, sbx.Commands.SendStdin(context.Background(), 3, "input\n"))
	assert.Equal(t, []byte("input\n"), input.GetInput().GetStdin())
	assert.EqualValues(t, 3, input.GetProcess().GetPid())

	require.NoError(t, sbx.Commands.CloseStdin(context.Background(), 3))
	assert.EqualValues(t, 3, closed.GetProcess().GetPid())
}

func TestRPCErrorsMapToTheSameTypesAsHTTP(t *testing.T) {
	// The point of errdefs.FromConnect: a caller classifies an envd failure
	// the same way it classifies a control-plane one.
	tests := map[connect.Code]error{
		connect.CodeNotFound:          errdefs.ErrNotFound,
		connect.CodeInvalidArgument:   errdefs.ErrInvalidArgument,
		connect.CodePermissionDenied:  errdefs.ErrForbidden,
		connect.CodeUnauthenticated:   errdefs.ErrAuth,
		connect.CodeAlreadyExists:     errdefs.ErrConflict,
		connect.CodeResourceExhausted: errdefs.ErrRateLimit,
	}

	for code, want := range tests {
		t.Run(code.String(), func(t *testing.T) {
			sbx := newEnvdSandbox(t, &fakeProcess{
				list: func(context.Context, *connect.Request[process.ListRequest]) (*connect.Response[process.ListResponse], error) {
					return nil, connect.NewError(code, assertNotFound)
				},
			}, "0.4.0")

			_, err := sbx.Commands.List(context.Background())
			require.Error(t, err)
			assert.ErrorIs(t, err, want)
		})
	}
}
