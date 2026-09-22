package sandbox

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process"
)

// SendStdin writes to a process's stdin. The process must have been started
// with CommandOptions.Stdin set, or there is nothing listening.
func (c *Commands) SendStdin(ctx context.Context, pid int, data string) error {
	req := &process.SendInputRequest{
		Process: selectorForPID(pid),
		Input: &process.ProcessInput{
			Input: &process.ProcessInput_Stdin{Stdin: []byte(data)},
		},
	}

	if _, err := c.sbx.conn.process.SendInput(ctx, requestFor(req, c.sbx, "")); err != nil {
		return errdefsFromConnect(err)
	}
	return nil
}
