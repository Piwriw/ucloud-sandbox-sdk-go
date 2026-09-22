package template

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// decodeBuildBody renders a builder and decodes the request body it would send.
func decodeBuildBody(t *testing.T, b *Builder) map[string]any {
	t.Helper()

	encoded, err := b.ToJSON()
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(encoded), &body))
	return body
}

func TestDefaultBaseImageFollowsRegion(t *testing.T) {
	// The Python SDK sends fromImage with a region-specific base image. The Go
	// SDK used to send fromTemplate:"base" instead -- a different field naming
	// a different thing. These assertions are what keeps the two aligned.
	tests := map[string]string{
		"cn-wlcb": CNBaseImage,
		"cn-sh":   CNBaseImage,
		"gray-1":  CNBaseImage,
		"us-ca":   DefaultBaseImage,
		"eu-fra":  DefaultBaseImage,
	}

	for region, wantImage := range tests {
		t.Run(region, func(t *testing.T) {
			body := decodeBuildBody(t, New(BuilderOptions{Region: region}))

			assert.Equal(t, wantImage, body["fromImage"])
			assert.NotContains(t, body, "fromTemplate",
				"a default base is an image, not a template")
		})
	}
}

func TestDefaultRegionFromEnvironment(t *testing.T) {
	t.Setenv(transport.EnvRegion, "us-ca")
	assert.Equal(t, DefaultBaseImage, decodeBuildBody(t, New(BuilderOptions{}))["fromImage"])

	t.Setenv(transport.EnvRegion, "cn-wlcb")
	assert.Equal(t, CNBaseImage, decodeBuildBody(t, New(BuilderOptions{}))["fromImage"])
}

func TestBaseImageOverride(t *testing.T) {
	b := New(BuilderOptions{Region: "cn-wlcb", BaseImage: "my/own:tag"})
	assert.Equal(t, "my/own:tag", decodeBuildBody(t, b)["fromImage"])
}

func TestFromBaseImageReturnsToTheRegionDefault(t *testing.T) {
	b := New(BuilderOptions{Region: "us-ca"}).
		FromImage("ubuntu:22.04").
		FromBaseImage()

	assert.Equal(t, DefaultBaseImage, decodeBuildBody(t, b)["fromImage"])
}

func TestFromTemplateClearsTheImage(t *testing.T) {
	body := decodeBuildBody(t, New(BuilderOptions{}).FromTemplate("my-base"))

	assert.Equal(t, "my-base", body["fromTemplate"])
	assert.NotContains(t, body, "fromImage")
}

func TestFromImageWithAuth(t *testing.T) {
	b := New(BuilderOptions{}).FromImageWithAuth("private.io/img:1", BasicAuth("user", "pass"))
	body := decodeBuildBody(t, b)

	assert.Equal(t, "private.io/img:1", body["fromImage"])
	assert.Equal(t, map[string]any{
		"type":     "registry",
		"username": "user",
		"password": "pass",
	}, body["fromImageRegistry"])
}

func TestRegistryVariants(t *testing.T) {
	t.Run("aws", func(t *testing.T) {
		b := New(BuilderOptions{}).FromImageWithAuth("x", AWSAuth("key", "secret", "us-east-1"))
		assert.Equal(t, map[string]any{
			"type":               "aws",
			"awsAccessKeyId":     "key",
			"awsSecretAccessKey": "secret",
			"awsRegion":          "us-east-1",
		}, decodeBuildBody(t, b)["fromImageRegistry"])
	})

	t.Run("gcp", func(t *testing.T) {
		b := New(BuilderOptions{}).FromImageWithAuth("x", GCPAuth(`{"type":"service_account"}`))
		assert.Equal(t, map[string]any{
			"type":               "gcp",
			"serviceAccountJson": `{"type":"service_account"}`,
		}, decodeBuildBody(t, b)["fromImageRegistry"])
	})
}

func TestStepsAreOrderedAndTyped(t *testing.T) {
	b := New(BuilderOptions{}).
		RunCmd("apt-get update").
		SetWorkdir("/app").
		SetUser("root").
		SetEnvs(map[string]string{"B": "2", "A": "1"})

	body := decodeBuildBody(t, b)
	steps, ok := body["steps"].([]any)
	require.True(t, ok, "steps should be a list")
	require.Len(t, steps, 4)

	types := make([]string, len(steps))
	for i, step := range steps {
		types[i] = step.(map[string]any)["type"].(string)
	}
	assert.Equal(t, []string{"RUN", "WORKDIR", "USER", "ENV"}, types)

	// ENV keys are sorted so the same map always yields the same cache key.
	assert.Equal(t, []any{"A", "1", "B", "2"}, steps[3].(map[string]any)["args"])
}

func TestStartAndReadyCommands(t *testing.T) {
	b := New(BuilderOptions{}).SetStartCmd("python -m http.server 8000", WaitForPort(8000))
	body := decodeBuildBody(t, b)

	assert.Equal(t, "python -m http.server 8000", body["startCmd"])
	assert.Equal(t, "ss -tuln | grep :8000", body["readyCmd"])
}

func TestReadyCmdHelpers(t *testing.T) {
	assert.Equal(t, "[ -f /tmp/ready ]", WaitForFile("/tmp/ready").Cmd)
	assert.Equal(t, "sleep 2.500", WaitForTimeout(2500).Cmd)
	assert.Equal(t, "sleep 1.000", WaitForTimeout(10).Cmd,
		"a sub-second wait is raised to one second")
}

func TestSkipCacheMarksOnlyTheNextStep(t *testing.T) {
	b := New(BuilderOptions{}).
		RunCmd("first").
		SkipCache().
		RunCmd("second").
		RunCmd("third")

	steps := decodeBuildBody(t, b)["steps"].([]any)
	require.Len(t, steps, 3)

	assert.NotContains(t, steps[0].(map[string]any), "force")
	assert.Equal(t, true, steps[1].(map[string]any)["force"])
	assert.NotContains(t, steps[2].(map[string]any), "force")
}

func TestNonBuildableStepsAreNotSent(t *testing.T) {
	// ENTRYPOINT arrives from a Dockerfile and becomes startCmd; sending it as
	// a step as well would be rejected by the build API.
	dir := t.TempDir()
	path := filepath.Join(dir, "Dockerfile")
	require.NoError(t, os.WriteFile(path, []byte("FROM ubuntu:22.04\nRUN echo hi\nENTRYPOINT /run.sh\n"), 0o644))

	b, err := FromDockerfile(path, BuilderOptions{})
	require.NoError(t, err)

	body := decodeBuildBody(t, b)
	steps := body["steps"].([]any)
	require.Len(t, steps, 1)
	assert.Equal(t, "RUN", steps[0].(map[string]any)["type"])
	assert.Equal(t, "/run.sh", body["startCmd"])
}

func TestFromDockerfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Dockerfile")
	require.NoError(t, os.WriteFile(path, []byte(`
FROM python:3.12
WORKDIR /app
ENV APP_ENV=prod
RUN pip install flask
USER app
CMD python app.py
`), 0o644))

	b, err := FromDockerfile(path, BuilderOptions{})
	require.NoError(t, err)

	body := decodeBuildBody(t, b)
	assert.Equal(t, "python:3.12", body["fromImage"])
	assert.Equal(t, "python app.py", body["startCmd"])

	steps := body["steps"].([]any)
	types := make([]string, len(steps))
	for i, step := range steps {
		types[i] = step.(map[string]any)["type"].(string)
	}
	assert.Equal(t, []string{"WORKDIR", "ENV", "RUN", "USER"}, types)

	// The Dockerfile's directory becomes the file context, so COPY sources
	// resolve next to it.
	assert.Equal(t, dir, b.fileContextPath)
}

func TestFromDockerfileBaseIsThePlatformImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Dockerfile")
	require.NoError(t, os.WriteFile(path, []byte("FROM base\nRUN echo hi\n"), 0o644))

	b, err := FromDockerfile(path, BuilderOptions{Region: "us-ca"})
	require.NoError(t, err)

	// "FROM base" names the platform's base image, not an image called "base"
	// on Docker Hub.
	assert.Equal(t, DefaultBaseImage, decodeBuildBody(t, b)["fromImage"])
}

func TestFromDockerfileRejectsMultipleFrom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Dockerfile")
	require.NoError(t, os.WriteFile(path, []byte("FROM a\nFROM b\n"), 0o644))

	_, err := FromDockerfile(path, BuilderOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple FROM")
}

func TestToDockerfileRoundTrip(t *testing.T) {
	b := New(BuilderOptions{}).
		FromImage("ubuntu:22.04").
		RunCmd("apt-get update").
		SetWorkdir("/app").
		SetEnvs(map[string]string{"A": "1"}).
		SetUser("app").
		SetStartCmd("/run.sh", ReadyCmd{})

	got, err := b.ToDockerfile()
	require.NoError(t, err)

	assert.Equal(t, `FROM ubuntu:22.04
RUN apt-get update
WORKDIR /app
ENV A=1
USER app
ENTRYPOINT /run.sh
`, got)
}

func TestToDockerfileRejectsTemplateBase(t *testing.T) {
	_, err := New(BuilderOptions{}).FromTemplate("other").ToDockerfile()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Dockerfile equivalent")
}
