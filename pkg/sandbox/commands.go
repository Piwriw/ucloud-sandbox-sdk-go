package sandbox

import (
	"strings"
	"sync"

	"connectrpc.com/connect"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process"
)

// Commands runs and manages processes inside a sandbox. Reach it through
// Sandbox.Commands.
type Commands struct {
	sbx *Sandbox
}

// shellCommand wraps a command line so envd runs it through a shell, which is
// what makes pipes, redirection and globbing work.
func shellCommand(cmd string) *process.ProcessConfig {
	return &process.ProcessConfig{
		Cmd:  "/bin/bash",
		Args: []string{"-l", "-c", cmd},
	}
}

// processConfig builds the process description for a command.
func (c *Commands) processConfig(cmd string, opts CommandOptions) *process.ProcessConfig {
	config := shellCommand(cmd)
	config.Envs = opts.EnvVars
	if opts.Cwd != "" {
		config.Cwd = &opts.Cwd
	}
	return config
}

// selectorForPID names a process by its PID.
func selectorForPID(pid int) *process.ProcessSelector {
	return &process.ProcessSelector{
		Selector: &process.ProcessSelector_Pid{Pid: uint32(pid)},
	}
}

// CommandHandle is a running command. Wait for it, feed it input, or kill it.
type CommandHandle struct {
	// PID is the process ID inside the sandbox.
	PID int

	cmds *Commands

	mu     sync.Mutex
	stdout strings.Builder
	stderr strings.Builder

	done   chan struct{}
	result *CommandResult
	err    error
}

// Stdout returns everything written to stdout so far.
func (h *CommandHandle) Stdout() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.stdout.String()
}

// Stderr returns everything written to stderr so far.
func (h *CommandHandle) Stderr() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.stderr.String()
}

// appendOutput records a chunk of output and hands it to the callback.
//
// The callback runs while the lock is not held, so a slow or re-entrant
// callback cannot block a reader calling Stdout.
func (h *CommandHandle) appendOutput(builder *strings.Builder, chunk []byte, callback func(string)) {
	text := string(chunk)

	h.mu.Lock()
	builder.WriteString(text)
	h.mu.Unlock()

	if callback != nil {
		callback(text)
	}
}

// errFromEnd builds the result of a finished process.
func errFromEnd(end *process.ProcessEvent_EndEvent, stdout, stderr string) *CommandResult {
	result := &CommandResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: int(end.GetExitCode()),
	}
	if end.Error != nil {
		result.Error = *end.Error
	}
	return result
}

// requestFor wraps a message with the acting user's header.
func requestFor[T any](msg *T, sbx *Sandbox, user string) *connect.Request[T] {
	return withUser(connect.NewRequest(msg), sbx.resolveUser(user))
}
