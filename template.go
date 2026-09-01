package sandbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	dockerfileparser "github.com/moby/buildkit/frontend/dockerfile/parser"
)

type InstructionType string

const (
	InstructionFrom       InstructionType = "FROM"
	InstructionCopy       InstructionType = "COPY"
	InstructionAdd        InstructionType = "ADD"
	InstructionRun        InstructionType = "RUN"
	InstructionEnv        InstructionType = "ENV"
	InstructionArg        InstructionType = "ARG"
	InstructionWorkdir    InstructionType = "WORKDIR"
	InstructionUser       InstructionType = "USER"
	InstructionEntrypoint InstructionType = "ENTRYPOINT"
	InstructionCmd        InstructionType = "CMD"
)

type RegistryConfig struct {
	Type               string `json:"type"`
	Username           string `json:"username,omitempty"`
	Password           string `json:"password,omitempty"`
	AWSAccessKeyID     string `json:"awsAccessKeyId,omitempty"`
	AWSSecretAccessKey string `json:"awsSecretAccessKey,omitempty"`
	AWSRegion          string `json:"awsRegion,omitempty"`
	ServiceAccountJSON string `json:"serviceAccountJson,omitempty"`
}

type Instruction struct {
	Type            InstructionType `json:"type"`
	Args            []string        `json:"args"`
	Force           bool            `json:"force,omitempty"`
	ForceUpload     *bool           `json:"forceUpload,omitempty"`
	ResolveSymlinks *bool           `json:"resolveSymlinks,omitempty"`
	FilesHash       string          `json:"filesHash,omitempty"`
}

type ReadyCmd struct {
	Cmd string
}

func (r ReadyCmd) String() string { return r.Cmd }

func WaitForPort(port int) ReadyCmd {
	return ReadyCmd{Cmd: fmt.Sprintf("ss -tuln | grep :%d", port)}
}

func WaitForFile(filename string) ReadyCmd {
	return ReadyCmd{Cmd: fmt.Sprintf("[ -f %s ]", filename)}
}

func WaitForTimeout(timeoutMs int) ReadyCmd {
	if timeoutMs < 1000 {
		timeoutMs = 1000
	}
	return ReadyCmd{Cmd: fmt.Sprintf("sleep %.3f", float64(timeoutMs)/1000.0)}
}

type registryConfig struct {
	username string
	password string
}

type RegistryOption func(*registryConfig)

func WithRegistryAuth(username, password string) RegistryOption {
	return func(c *registryConfig) {
		c.username = username
		c.password = password
	}
}

func applyRegistryOpts(opts []RegistryOption) *registryConfig {
	cfg := &registryConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

type TemplateBuilder struct {
	baseImage       string
	baseTemplate    string
	registryConfig  *RegistryConfig
	startCmd        string
	readyCmd        string
	force           bool
	forceNextLayer  bool
	instructions    []Instruction
	fileContextPath string
	ignorePatterns  []string
	err             error
}

type TemplateBuilderOption func(*TemplateBuilder)

func WithFileContextPath(path string) TemplateBuilderOption {
	return func(t *TemplateBuilder) { t.fileContextPath = path }
}

func NewTemplate(opts ...TemplateBuilderOption) *TemplateBuilder {
	t := &TemplateBuilder{baseTemplate: DefaultTemplate}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

func (t *TemplateBuilder) FromImage(image string, opts ...RegistryOption) *TemplateBuilder {
	t.baseImage = image
	t.baseTemplate = ""
	cfg := applyRegistryOpts(opts)
	if cfg.username != "" && cfg.password != "" {
		t.registryConfig = &RegistryConfig{Type: "registry", Username: cfg.username, Password: cfg.password}
	} else {
		t.registryConfig = nil
	}
	if t.forceNextLayer {
		t.force = true
	}
	return t
}

// From sets the base image used by the template.
func (t *TemplateBuilder) From(image string, opts ...RegistryOption) *TemplateBuilder {
	return t.FromImage(image, opts...)
}

func (t *TemplateBuilder) FromTemplate(template string) *TemplateBuilder {
	t.baseTemplate = template
	t.baseImage = ""
	t.registryConfig = nil
	if t.forceNextLayer {
		t.force = true
	}
	return t
}

func (t *TemplateBuilder) FromBaseImage() *TemplateBuilder {
	return t.FromTemplate(DefaultTemplate)
}

func (t *TemplateBuilder) addInstruction(inst Instruction) {
	inst.Force = t.forceNextLayer
	t.instructions = append(t.instructions, inst)
	t.forceNextLayer = false
}

func (t *TemplateBuilder) RunCmd(command string) *TemplateBuilder {
	t.addInstruction(Instruction{Type: InstructionRun, Args: []string{command}})
	return t
}

func (t *TemplateBuilder) RunCmdAsUser(command, user string) *TemplateBuilder {
	args := []string{command}
	if user != "" {
		args = append(args, user)
	}
	t.addInstruction(Instruction{Type: InstructionRun, Args: args})
	return t
}

func (t *TemplateBuilder) SetEnvs(envs map[string]string) *TemplateBuilder {
	if len(envs) == 0 {
		return t
	}
	keys := make([]string, 0, len(envs))
	for k := range envs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	args := make([]string, 0, len(envs)*2)
	for _, k := range keys {
		args = append(args, k, envs[k])
	}
	t.addInstruction(Instruction{Type: InstructionEnv, Args: args})
	return t
}

func (t *TemplateBuilder) SetWorkdir(workdir string) *TemplateBuilder {
	t.addInstruction(Instruction{Type: InstructionWorkdir, Args: []string{workdir}})
	return t
}

func (t *TemplateBuilder) SetUser(user string) *TemplateBuilder {
	t.addInstruction(Instruction{Type: InstructionUser, Args: []string{user}})
	return t
}

func (t *TemplateBuilder) SkipCache() *TemplateBuilder {
	t.forceNextLayer = true
	return t
}

func (t *TemplateBuilder) SetStartCmd(startCmd string, readyCmd ReadyCmd) *TemplateBuilder {
	t.startCmd = startCmd
	t.readyCmd = readyCmd.Cmd
	return t
}

func (t *TemplateBuilder) serialize() templateData {
	data := templateData{
		Steps: t.instructions,
		Force: t.force,
	}
	if t.baseImage != "" {
		data.FromImage = t.baseImage
	}
	if t.baseTemplate != "" {
		data.FromTemplate = t.baseTemplate
	}
	if t.registryConfig != nil {
		data.FromImageRegistry = t.registryConfig
	}
	if t.startCmd != "" {
		data.StartCmd = t.startCmd
	}
	if t.readyCmd != "" {
		data.ReadyCmd = t.readyCmd
	}
	return data
}

func (t *TemplateBuilder) ToJSON() (string, error) {
	data := t.serialize()
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (t *TemplateBuilder) ToDockerfile() (string, error) {
	if t.baseTemplate != "" {
		return "", fmt.Errorf("cannot convert template built from another template to Dockerfile")
	}
	if t.baseImage == "" {
		return "", fmt.Errorf("no base image specified for template")
	}

	var buf strings.Builder
	buf.WriteString("FROM ")
	buf.WriteString(t.baseImage)
	buf.WriteString("\n")

	for _, inst := range t.instructions {
		switch inst.Type {
		case InstructionRun:
			if len(inst.Args) == 0 {
				continue
			}
			if len(inst.Args) > 1 && inst.Args[1] != "" {
				buf.WriteString("USER ")
				buf.WriteString(inst.Args[1])
				buf.WriteString("\n")
			}
			buf.WriteString("RUN ")
			buf.WriteString(inst.Args[0])
			buf.WriteString("\n")
		case InstructionCopy:
			if len(inst.Args) >= 2 {
				buf.WriteString("COPY ")
				buf.WriteString(inst.Args[0])
				buf.WriteString(" ")
				buf.WriteString(inst.Args[1])
				buf.WriteString("\n")
			}
		case InstructionAdd:
			if len(inst.Args) >= 2 {
				buf.WriteString("ADD ")
				buf.WriteString(strings.Join(inst.Args, " "))
				buf.WriteString("\n")
			}
		case InstructionArg:
			if len(inst.Args) > 0 {
				buf.WriteString("ARG ")
				buf.WriteString(inst.Args[0])
				if len(inst.Args) > 1 {
					buf.WriteString("=")
					buf.WriteString(inst.Args[1])
				}
				buf.WriteString("\n")
			}
		case InstructionEnv:
			var pairs []string
			for i := 0; i+1 < len(inst.Args); i += 2 {
				pairs = append(pairs, inst.Args[i]+"="+inst.Args[i+1])
			}
			if len(pairs) > 0 {
				buf.WriteString("ENV ")
				buf.WriteString(strings.Join(pairs, " "))
				buf.WriteString("\n")
			}
		case InstructionWorkdir:
			if len(inst.Args) > 0 {
				buf.WriteString("WORKDIR ")
				buf.WriteString(inst.Args[0])
				buf.WriteString("\n")
			}
		case InstructionUser:
			if len(inst.Args) > 0 {
				buf.WriteString("USER ")
				buf.WriteString(inst.Args[0])
				buf.WriteString("\n")
			}
		default:
			buf.WriteString(string(inst.Type))
			buf.WriteString(" ")
			buf.WriteString(strings.Join(inst.Args, " "))
			buf.WriteString("\n")
		}
	}

	if t.startCmd != "" {
		buf.WriteString("ENTRYPOINT ")
		buf.WriteString(t.startCmd)
		buf.WriteString("\n")
	}

	return buf.String(), nil
}

// FromDockerfile creates a template builder from a Dockerfile.
func FromDockerfile(path string, opts ...TemplateBuilderOption) (*TemplateBuilder, error) {
	return NewTemplate(opts...).FromDockerfile(path)
}

// FromDockerfile adds Dockerfile instructions to the builder.
func (t *TemplateBuilder) FromDockerfile(path string) (*TemplateBuilder, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return t, fmt.Errorf("read Dockerfile %q: %w", path, err)
	}

	parsed := NewTemplate(WithFileContextPath(filepath.Dir(path)))
	if err := parsed.parseDockerfile(bytes.NewReader(content)); err != nil {
		return t, fmt.Errorf("parse Dockerfile %q: %w", path, err)
	}

	t.baseImage = parsed.baseImage
	t.baseTemplate = parsed.baseTemplate
	t.registryConfig = parsed.registryConfig
	t.startCmd = parsed.startCmd
	t.readyCmd = parsed.readyCmd
	t.instructions = parsed.instructions
	t.fileContextPath = parsed.fileContextPath
	return t, nil
}

// parseDockerfile parses Dockerfile content into the template builder.
func (t *TemplateBuilder) parseDockerfile(r io.Reader) error {
	result, err := dockerfileparser.Parse(r)
	if err != nil {
		return err
	}

	seenFrom := false
	for _, node := range result.AST.Children {
		name := strings.ToUpper(node.Value)
		lineNo := node.StartLine
		args := dockerfileNodeArgs(node)
		switch InstructionType(name) {
		case InstructionFrom:
			if len(args) < 1 || (len(args) != 1 && !(len(args) == 3 && strings.EqualFold(args[1], "AS"))) {
				return fmt.Errorf("line %d: FROM requires exactly one image", lineNo)
			}
			if seenFrom {
				return fmt.Errorf("line %d: multiple FROM instructions are not supported", lineNo)
			}
			if strings.EqualFold(args[0], "base") {
				t.FromBaseImage()
			} else {
				t.FromImage(args[0])
			}
			seenFrom = true
		case InstructionRun:
			if len(args) == 0 {
				return fmt.Errorf("line %d: RUN requires a command", lineNo)
			}
			t.RunCmd(dockerfileCommand(args))
		case InstructionCopy, InstructionAdd:
			if len(args) < 2 {
				return fmt.Errorf("line %d: %s requires source and destination", lineNo, name)
			}
			for _, src := range args[:len(args)-1] {
				instArgs := []string{src, args[len(args)-1]}
				if user := dockerfileChown(node); user != "" {
					instArgs = append(instArgs, user)
				}
				t.addInstruction(Instruction{Type: InstructionCopy, Args: instArgs})
			}
		case InstructionEnv:
			if len(args) == 0 {
				return fmt.Errorf("line %d: invalid %s instruction", lineNo, name)
			}
			if InstructionType(name) == InstructionEnv && len(args) < 2 {
				return fmt.Errorf("line %d: invalid ENV instruction", lineNo)
			}
			envs, err := dockerfileEnvArgs(node, args)
			if err != nil {
				return fmt.Errorf("line %d: %w", lineNo, err)
			}
			t.SetEnvs(envs)
		case InstructionArg:
			if len(args) == 0 {
				return fmt.Errorf("line %d: invalid ARG instruction", lineNo)
			}
			for _, arg := range args {
				key, value, hasValue := strings.Cut(arg, "=")
				if key == "" || strings.ContainsAny(key, " \t") {
					return fmt.Errorf("line %d: invalid ARG assignment", lineNo)
				}
				instructionArgs := []string{key}
				if hasValue {
					instructionArgs = append(instructionArgs, value)
				}
				t.addInstruction(Instruction{Type: InstructionArg, Args: instructionArgs})
			}

		case InstructionWorkdir:
			if len(args) == 0 {
				return fmt.Errorf("line %d: WORKDIR requires a path", lineNo)
			}
			t.SetWorkdir(args[0])
		case InstructionUser:
			if len(args) == 0 {
				return fmt.Errorf("line %d: USER requires a user", lineNo)
			}
			t.SetUser(args[0])
		case InstructionEntrypoint:
			if len(args) == 0 {
				return fmt.Errorf("line %d: ENTRYPOINT requires a command", lineNo)
			}
			t.SetStartCmd(dockerfileCommand(args), ReadyCmd{})
		case InstructionCmd:
			if len(args) == 0 {
				return fmt.Errorf("line %d: CMD requires a command", lineNo)
			}
			t.SetStartCmd(dockerfileCommand(args), ReadyCmd{})
		default:
			log.Printf("warning: unsupported Dockerfile instruction %q on line %d", name, lineNo)
		}
	}
	if !seenFrom {
		return fmt.Errorf("FROM instruction is required")
	}
	return nil
}

// dockerfileNodeArgs returns the arguments from a parsed Dockerfile node.
func dockerfileNodeArgs(node *dockerfileparser.Node) []string {
	var args []string
	for arg := node.Next; arg != nil; arg = arg.Next {
		args = append(args, arg.Value)
	}
	return args
}

// dockerfileEnvArgs parses ENV or ARG arguments into environment values.
func dockerfileEnvArgs(node *dockerfileparser.Node, args []string) (map[string]string, error) {
	envs := make(map[string]string)
	if strings.EqualFold(node.Value, "ARG") {
		for _, arg := range args {
			key, value, hasValue := strings.Cut(arg, "=")
			if key == "" || strings.ContainsAny(key, " \t") {
				return nil, fmt.Errorf("invalid ARG assignment")
			}
			if !hasValue {
				value = ""
			}
			envs[key] = value
		}
		return envs, nil
	}

	// BuildKit represents ENV key=value pairs as key, value, "=" triples.
	if len(args) == 3 && args[2] == "" {
		envs[args[0]] = args[1]
		return envs, nil
	}
	if len(args) == 2 {
		envs[args[0]] = args[1]
		return envs, nil
	}
	if len(args)%3 != 0 {
		return nil, fmt.Errorf("invalid ENV assignment")
	}
	for i := 0; i < len(args); i += 3 {
		if args[i] == "" || args[i+2] != "=" {
			return nil, fmt.Errorf("invalid ENV assignment")
		}
		envs[args[i]] = args[i+1]
	}
	return envs, nil
}

// dockerfileCommand joins parsed Dockerfile command arguments.
func dockerfileCommand(args []string) string {
	return strings.Join(args, " ")
}

// dockerfileChown returns the chown value from a Dockerfile instruction flag.
func dockerfileChown(node *dockerfileparser.Node) string {
	for _, flag := range node.Flags {
		if strings.HasPrefix(strings.ToLower(flag), "--chown=") {
			return flag[len("--chown="):]
		}
	}
	return ""
}
