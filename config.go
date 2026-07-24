package sandbox

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ConnectionConfig holds API endpoint and transport settings for Client.
type ConnectionConfig struct {
	APIKey          string
	Domain          string
	APIURL          string
	Debug           bool
	RequestTimeout  time.Duration
	Headers         map[string]string
	SandboxURL      string
	AccessToken     string
	Logger          Logger
	InsecureSkipTLS bool
	InsecureHTTP    bool
}

func (c *ConnectionConfig) GetHost(sandboxID, sandboxDomain string, port int) string {
	if c.Debug {
		return fmt.Sprintf("localhost:%d", port)
	}
	return fmt.Sprintf("%d-%s.%s", port, sandboxID, sandboxDomain)
}

func (c *ConnectionConfig) GetSandboxURL(sandboxID, sandboxDomain string) string {
	if c.SandboxURL != "" {
		return c.SandboxURL
	}
	scheme := "https"
	if c.Debug || c.InsecureHTTP {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s", scheme, c.GetHost(sandboxID, sandboxDomain, EnvdPort))
}

func versionLessThan(a, b [3]int) bool {
	if a[0] != b[0] {
		return a[0] < b[0]
	}
	if a[1] != b[1] {
		return a[1] < b[1]
	}
	return a[2] < b[2]
}

func parseEnvdVersion(version string) [3]int {
	var v [3]int
	parts := strings.Split(version, ".")
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err == nil {
			v[i] = n
		}
	}
	return v
}
