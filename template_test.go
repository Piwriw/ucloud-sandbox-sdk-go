package sandbox

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseDockerfile(t *testing.T) {
	content := `# base image
FROM golang:1.22

ENV APP_ENV=production PORT=8080
WORKDIR /app
COPY [".", "/app"]
RUN go build -o server .
USER app
ENTRYPOINT ["./server"]
CMD ["--port", "8080"]
`

	builder := NewTemplate()
	if err := builder.parseDockerfile(strings.NewReader(content)); err != nil {
		t.Fatalf("parseDockerfile() error = %v", err)
	}

	if builder.baseImage != "golang:1.22" {
		t.Fatalf("baseImage = %q, want %q", builder.baseImage, "golang:1.22")
	}
	want := []Instruction{
		{Type: InstructionEnv, Args: []string{"APP_ENV", "production", "PORT", "8080"}},
		{Type: InstructionWorkdir, Args: []string{"/app"}},
		{Type: InstructionCopy, Args: []string{".", "/app"}},
		{Type: InstructionRun, Args: []string{"go build -o server ."}},
		{Type: InstructionUser, Args: []string{"app"}},
		{Type: InstructionEntrypoint, Args: []string{"./server"}},
		{Type: InstructionCmd, Args: []string{"--port 8080"}},
	}
	if !reflect.DeepEqual(builder.instructions, want) {
		t.Errorf("instructions = %#v, want %#v", builder.instructions, want)
	}
}

func TestParseDockerfileSupportsContinuationAndCaseInsensitiveInstructions(t *testing.T) {
	builder := NewTemplate()
	content := "from alpine:3.20\n" +
		"run apk add --no-cache \\\n\t\tca-certificates\n" +
		"entrypoint /bin/sh\n" +
		"cmd -c 'echo ready'\n"

	if err := builder.parseDockerfile(strings.NewReader(content)); err != nil {
		t.Fatalf("parseDockerfile() error = %v", err)
	}

	if got, want := builder.baseImage, "alpine:3.20"; got != want {
		t.Errorf("baseImage = %q, want %q", got, want)
	}
	if got, want := builder.instructions[0].Args[0], "apk add --no-cache \t\tca-certificates"; got != want {
		t.Errorf("continued RUN command = %q, want %q", got, want)
	}
	if got, want := builder.instructions[1].Type, InstructionEntrypoint; got != want {
		t.Errorf("entrypoint type = %q, want %q", got, want)
	}
	if got, want := builder.instructions[2].Type, InstructionCmd; got != want {
		t.Errorf("cmd type = %q, want %q", got, want)
	}
}

func TestParseDockerfileParsesShellFormsAndSortsEnvironmentVariables(t *testing.T) {
	content := "FROM alpine:3.20\n" +
		"ENV APP_ENV production api server\n" +
		"env ZED=last ALPHA=first\n" +
		"copy app /app\n" +
		"WORKDIR /app\n" +
		"USER 1000:1000\n" +
		"RUN echo ready\n" +
		"ENTRYPOINT [\"/bin/sh\", \"-c\"]\n" +
		"CMD [\"echo\", \"hello\"]\n"

	builder := NewTemplate()
	if err := builder.parseDockerfile(strings.NewReader(content)); err != nil {
		t.Fatalf("parseDockerfile() error = %v", err)
	}

	want := []Instruction{
		{Type: InstructionEnv, Args: []string{"APP_ENV", "production api server"}},
		{Type: InstructionEnv, Args: []string{"ALPHA", "first", "ZED", "last"}},
		{Type: InstructionCopy, Args: []string{"app", "/app"}},
		{Type: InstructionWorkdir, Args: []string{"/app"}},
		{Type: InstructionUser, Args: []string{"1000:1000"}},
		{Type: InstructionRun, Args: []string{"echo ready"}},
		{Type: InstructionEntrypoint, Args: []string{"/bin/sh -c"}},
		{Type: InstructionCmd, Args: []string{"echo hello"}},
	}
	if !reflect.DeepEqual(builder.instructions, want) {
		t.Errorf("instructions = %#v, want %#v", builder.instructions, want)
	}
}

func TestParseDockerfileSupportsAddAndArg(t *testing.T) {
	content := "FROM alpine:3.20\n" +
		"ARG BUILD_VERSION=1.2.3\n" +
		"ARG EMPTY_VERSION\n" +
		"ADD [\"app\", \"config\", \"/opt/app/\"]\n"

	builder := NewTemplate()
	if err := builder.parseDockerfile(strings.NewReader(content)); err != nil {
		t.Fatalf("parseDockerfile() error = %v", err)
	}

	want := []Instruction{
		{Type: InstructionArg, Args: []string{"BUILD_VERSION", "1.2.3"}},
		{Type: InstructionArg, Args: []string{"EMPTY_VERSION"}},
		{Type: InstructionCopy, Args: []string{"app", "/opt/app/"}},
		{Type: InstructionCopy, Args: []string{"config", "/opt/app/"}},
	}
	if !reflect.DeepEqual(builder.instructions, want) {
		t.Errorf("instructions = %#v, want %#v", builder.instructions, want)
	}
}

func TestParseDockerfileAddChown(t *testing.T) {
	builder := NewTemplate()
	if err := builder.parseDockerfile(strings.NewReader("FROM alpine\nADD --chown=1000:1000 app /app\n")); err != nil {
		t.Fatalf("parseDockerfile() error = %v", err)
	}

	want := []Instruction{{Type: InstructionCopy, Args: []string{"app", "/app", "1000:1000"}}}
	if !reflect.DeepEqual(builder.instructions, want) {
		t.Errorf("instructions = %#v, want %#v", builder.instructions, want)
	}
}

func TestToDockerfileNormalizesAddAndArg(t *testing.T) {
	builder := NewTemplate()
	if err := builder.parseDockerfile(strings.NewReader("FROM alpine\nARG VERSION=1\nADD app /app\n")); err != nil {
		t.Fatalf("parseDockerfile() error = %v", err)
	}

	got, err := builder.ToDockerfile()
	if err != nil {
		t.Fatalf("ToDockerfile() error = %v", err)
	}
	want := "FROM alpine\nARG VERSION=1\nCOPY app /app\n"
	if got != want {
		t.Errorf("ToDockerfile() = %q, want %q", got, want)
	}
}

func TestParseDockerfileRejectsInvalidAddAndArg(t *testing.T) {
	for _, tt := range []struct {
		name, content, want string
	}{
		{"empty ADD", "FROM alpine\nADD src\n", "ADD requires source and destination"},
		{"empty ARG", "FROM alpine\nARG\n", "invalid ARG instruction"},
		{"invalid ARG assignment", "FROM alpine\nARG =value\n", "invalid ARG assignment"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := NewTemplate().parseDockerfile(strings.NewReader(tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("parseDockerfile() error = %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestParseDockerfileErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "missing from",
			content: "RUN echo hello\n",
			wantErr: "FROM instruction is required",
		},
		{
			name:    "comments only",
			content: "\n  # no instructions\n\t# still no FROM\n",
			wantErr: "file with no instructions",
		},
		{
			name:    "multiple from",
			content: "FROM alpine\nFROM ubuntu\n",
			wantErr: "multiple FROM instructions are not supported",
		},
		{
			name:    "from without image",
			content: "FROM\n",
			wantErr: "FROM requires exactly one image",
		},
		{
			name:    "from with multiple images",
			content: "FROM alpine ubuntu\n",
			wantErr: "FROM requires exactly one image",
		},
		{
			name:    "empty run",
			content: "FROM alpine\nRUN\n",
			wantErr: "RUN requires a command",
		},
		{
			name:    "invalid copy JSON",
			content: "FROM alpine\nCOPY [\"src\"]\n",
			wantErr: "COPY requires source and destination",
		},
		{
			name:    "empty env",
			content: "FROM alpine\nENV\n",
			wantErr: "invalid ENV instruction",
		},
		{
			name:    "empty env key",
			content: "FROM alpine\nENV =value OTHER=foo\n",
			wantErr: "invalid ENV assignment",
		},
		{
			name:    "empty workdir",
			content: "FROM alpine\nWORKDIR\n",
			wantErr: "WORKDIR requires a path",
		},
		{
			name:    "empty user",
			content: "FROM alpine\nUSER\n",
			wantErr: "USER requires a user",
		},
		{
			name:    "empty entrypoint",
			content: "FROM alpine\nENTRYPOINT\n",
			wantErr: "ENTRYPOINT requires a command",
		},
		{
			name:    "empty cmd",
			content: "FROM alpine\nCMD\n",
			wantErr: "CMD requires a command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewTemplate().parseDockerfile(strings.NewReader(tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("parseDockerfile() error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestFromDockerfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Dockerfile")
	content := "FROM alpine:3.20\nCOPY app /app\n"
	if err := writeTestFile(path, content); err != nil {
		t.Fatal(err)
	}

	builder, err := FromDockerfile(path)
	if err != nil {
		t.Fatalf("FromDockerfile() error = %v", err)
	}
	if got, want := builder.fileContextPath, dir; got != want {
		t.Errorf("fileContextPath = %q, want %q", got, want)
	}
	if got, want := builder.baseImage, "alpine:3.20"; got != want {
		t.Errorf("baseImage = %q, want %q", got, want)
	}
}

func TestParseDockerfileErrorIncludesLogicalLineNumber(t *testing.T) {
	content := "# comment\n\nFROM alpine\n\nRUN\n"
	err := NewTemplate().parseDockerfile(strings.NewReader(content))
	if err == nil || !strings.Contains(err.Error(), "line 5: RUN requires a command") {
		t.Fatalf("parseDockerfile() error = %v, want logical line number and message", err)
	}
}

func TestParseDockerfileSupportsFromStageAndFlags(t *testing.T) {
	content := "FROM --platform=linux/amd64 golang:1.22 AS builder\n" +
		"COPY --chown=app:app go.mod go.sum /src/\n" +
		"RUN go build ./...\n"

	builder := NewTemplate()
	if err := builder.parseDockerfile(strings.NewReader(content)); err != nil {
		t.Fatalf("parseDockerfile() error = %v", err)
	}

	if got, want := builder.baseImage, "golang:1.22"; got != want {
		t.Errorf("baseImage = %q, want %q", got, want)
	}
	want := []Instruction{
		{Type: InstructionCopy, Args: []string{"go.mod", "/src/", "app:app"}},
		{Type: InstructionCopy, Args: []string{"go.sum", "/src/", "app:app"}},
		{Type: InstructionRun, Args: []string{"go build ./..."}},
	}
	if !reflect.DeepEqual(builder.instructions, want) {
		t.Errorf("instructions = %#v, want %#v", builder.instructions, want)
	}
}

func TestParseDockerfilePreservesQuotedValuesAndCommands(t *testing.T) {
	content := "FROM alpine\n" +
		"ENV GREETING=hello\\ world DESCRIPTION=\"a value with spaces\"\n" +
		"RUN printf '%s\\n' \"$GREETING\"\n" +
		"ENTRYPOINT [\"/bin/sh\", \"-c\", \"echo $GREETING\"]\n" +
		"CMD [\"--verbose\", \"true\"]\n"

	builder := NewTemplate()
	if err := builder.parseDockerfile(strings.NewReader(content)); err != nil {
		t.Fatalf("parseDockerfile() error = %v", err)
	}

	want := []Instruction{
		{Type: InstructionEnv, Args: []string{"DESCRIPTION", "\"a value with spaces\"", "GREETING", "hello\\ world"}},
		{Type: InstructionRun, Args: []string{"printf '%s\\n' \"$GREETING\""}},
		{Type: InstructionEntrypoint, Args: []string{"/bin/sh -c echo $GREETING"}},
		{Type: InstructionCmd, Args: []string{"--verbose true"}},
	}
	if !reflect.DeepEqual(builder.instructions, want) {
		t.Errorf("instructions = %#v, want %#v", builder.instructions, want)
	}
}

func TestParseDockerfileIgnoresUnsupportedInstructions(t *testing.T) {
	content := "FROM alpine\nLABEL maintainer=team\nHEALTHCHECK CMD wget localhost\nRUN echo ready\n"
	builder := NewTemplate()
	if err := builder.parseDockerfile(strings.NewReader(content)); err != nil {
		t.Fatalf("parseDockerfile() error = %v", err)
	}

	want := []Instruction{{Type: InstructionRun, Args: []string{"echo ready"}}}
	if !reflect.DeepEqual(builder.instructions, want) {
		t.Errorf("instructions = %#v, want %#v", builder.instructions, want)
	}
}

func TestParseDockerfilePropagatesParserErrors(t *testing.T) {
	content := "FROM alpine\nRUN [1, 2]\n"
	err := NewTemplate().parseDockerfile(strings.NewReader(content))
	if err == nil {
		t.Fatal("parseDockerfile() error = nil, want parser error")
	}
	if strings.Contains(err.Error(), "FROM instruction is required") {
		t.Fatalf("parseDockerfile() error = %v, want parser error", err)
	}
}

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
