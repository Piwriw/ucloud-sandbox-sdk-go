package sandbox

import (
	"context"
	"encoding/base64"
	"net/http"
	"strconv"

	"connectrpc.com/connect"

	envdapi "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/filesystem/filesystemconnect"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/process/processconnect"
)

// envd's own headers. The access token authenticates to envd itself; the
// traffic token gets past the proxy in front of it; the sandbox ID and port
// tell that proxy which sandbox to route to.
const (
	headerAccessToken        = "X-Access-Token"
	headerTrafficAccessToken = "E2B-Traffic-Access-Token"
	headerSandboxID          = "E2b-Sandbox-Id"
	headerSandboxPort        = "E2b-Sandbox-Port"
	headerKeepalivePing      = "Keepalive-Ping-Interval"
)

// envdConn holds everything needed to reach one sandbox's envd.
type envdConn struct {
	baseURL string
	headers map[string]string

	process    processconnect.ProcessClient
	filesystem filesystemconnect.FilesystemClient
	files      *envdapi.ClientWithResponses
}

// newEnvdConn builds the clients for one sandbox's envd.
//
// The connect clients use the default protobuf codec. envd speaks the connect
// protocol natively, so no per-call encoding choices are needed here.
func newEnvdConn(httpClient *http.Client, baseURL, sandboxID, accessToken, trafficToken, apiKey string) (*envdConn, error) {
	headers := map[string]string{
		headerSandboxID:     sandboxID,
		headerSandboxPort:   strconv.Itoa(EnvdPort),
		headerKeepalivePing: strconv.Itoa(keepalivePingIntervalSec),
	}
	// A secured sandbox issues its own access token. Without one, the
	// account's API key is what envd will accept.
	if accessToken != "" {
		headers[headerAccessToken] = accessToken
	} else if apiKey != "" {
		headers["X-API-Key"] = apiKey
	}
	if trafficToken != "" {
		headers[headerTrafficAccessToken] = trafficToken
	}

	conn := &envdConn{baseURL: baseURL, headers: headers}

	interceptor := connect.WithInterceptors(connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				conn.applyHeaders(req.Header())
				return next(ctx, req)
			}
		}))

	conn.process = processconnect.NewProcessClient(httpClient, baseURL, interceptor)
	conn.filesystem = filesystemconnect.NewFilesystemClient(httpClient, baseURL, interceptor)

	filesClient, err := envdapi.NewClientWithResponses(
		baseURL,
		envdapi.WithHTTPClient(httpClient),
		envdapi.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			conn.applyHeaders(req.Header)
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}
	conn.files = filesClient

	return conn, nil
}

// applyHeaders adds envd's headers to an outgoing request.
func (c *envdConn) applyHeaders(header http.Header) {
	for key, value := range c.headers {
		header.Set(key, value)
	}
}

// userHeader returns the basic-auth header envd reads the acting user from, or
// nil when the sandbox's default user should be used.
//
// envd takes the user in the password-less half of a basic-auth credential,
// which is why this is not a plain header value.
func userHeader(user string) map[string]string {
	if user == "" {
		return nil
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(user + ":"))
	return map[string]string{"Authorization": "Basic " + encoded}
}

// withUser attaches the acting user to a connect request.
func withUser[T any](req *connect.Request[T], user string) *connect.Request[T] {
	for key, value := range userHeader(user) {
		req.Header().Set(key, value)
	}
	return req
}
