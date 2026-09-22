package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process"
)

// List returns the processes envd is tracking inside the sandbox. The result is
// never nil.
func (c *Commands) List(ctx context.Context) ([]ProcessInfo, error) {
	resp, err := c.sbx.conn.process.List(ctx, requestFor(&process.ListRequest{}, c.sbx, ""))
	if err != nil {
		return nil, errdefsFromConnect(err)
	}

	listed := resp.Msg.GetProcesses()
	processes := make([]ProcessInfo, 0, len(listed))
	for _, p := range listed {
		config := p.GetConfig()
		processes = append(processes, ProcessInfo{
			PID:  int(p.GetPid()),
			Tag:  p.GetTag(),
			Cmd:  config.GetCmd(),
			Args: config.GetArgs(),
			Envs: config.GetEnvs(),
			Cwd:  config.GetCwd(),
		})
	}
	return processes, nil
}
