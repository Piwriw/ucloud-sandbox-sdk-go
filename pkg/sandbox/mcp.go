package sandbox

import (
	"context"
	"fmt"
)

// mcpTokenPath is where the MCP gateway leaves its token inside the sandbox.
const mcpTokenPath = "/etc/mcp-gateway/.token"

// MCPURL returns the URL of the sandbox's MCP gateway.
func (s *Sandbox) MCPURL() string {
	return fmt.Sprintf("%s://%s/mcp", s.svc.t.SandboxScheme(), s.Host(MCPPort))
}

// MCPToken returns the token the MCP gateway expects.
//
// The token is read out of the sandbox once and kept, since it does not change
// for the sandbox's lifetime. Concurrent callers share one read.
func (s *Sandbox) MCPToken(ctx context.Context) (string, error) {
	s.mcpTokenOnce.Do(func() {
		s.mcpToken, s.mcpTokenErr = s.Files.Read(ctx, mcpTokenPath, FileOptions{User: "root"})
	})
	return s.mcpToken, s.mcpTokenErr
}
