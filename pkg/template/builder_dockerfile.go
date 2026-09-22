package template

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	dockerfileparser "github.com/moby/buildkit/frontend/dockerfile/parser"
)

// ToDockerfile renders the template as a Dockerfile.
//
// A template based on another template has no Dockerfile equivalent, and is
// rejected.
func (b *Builder) ToDockerfile() (string, error) {
	if b.err != nil {
		return "", b.err
	}
	if b.baseTemplate != "" {
		return "", fmt.Errorf("template: a template built from another template has no Dockerfile equivalent")
	}
	if b.baseImage == "" {
		return "", fmt.Errorf("template: no base image specified")
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "FROM %s\n", b.baseImage)

	for _, inst := range b.instructions {
		switch inst.Type {
		case InstructionRun:
			if len(inst.Args) == 0 {
				continue
			}
			// RunCmdAsUser stores the user as a second argument; a Dockerfile
			// expresses that as a USER line before the RUN.
			if len(inst.Args) > 1 && inst.Args[1] != "" {
				fmt.Fprintf(&buf, "USER %s\n", inst.Args[1])
			}
			fmt.Fprintf(&buf, "RUN %s\n", inst.Args[0])

		case InstructionCopy:
			if len(inst.Args) >= 2 {
				fmt.Fprintf(&buf, "COPY %s %s\n", inst.Args[0], inst.Args[1])
			}

		case InstructionAdd:
			if len(inst.Args) >= 2 {
				fmt.Fprintf(&buf, "ADD %s\n", strings.Join(inst.Args, " "))
			}

		case InstructionArg:
			if len(inst.Args) > 0 {
				buf.WriteString("ARG " + inst.Args[0])
				if len(inst.Args) > 1 {
					buf.WriteString("=" + inst.Args[1])
				}
				buf.WriteString("\n")
			}

		case InstructionEnv:
			var pairs []string
			for i := 0; i+1 < len(inst.Args); i += 2 {
				pairs = append(pairs, inst.Args[i]+"="+inst.Args[i+1])
			}
			if len(pairs) > 0 {
				fmt.Fprintf(&buf, "ENV %s\n", strings.Join(pairs, " "))
			}

		case InstructionWorkdir:
			if len(inst.Args) > 0 {
				fmt.Fprintf(&buf, "WORKDIR %s\n", inst.Args[0])
			}

		case InstructionUser:
			if len(inst.Args) > 0 {
				fmt.Fprintf(&buf, "USER %s\n", inst.Args[0])
			}

		default:
			fmt.Fprintf(&buf, "%s %s\n", inst.Type, strings.Join(inst.Args, " "))
		}
	}

	if b.startCmd != "" {
		fmt.Fprintf(&buf, "ENTRYPOINT %s\n", b.startCmd)
	}

	return buf.String(), nil
}

// FromDockerfile builds a template from a Dockerfile on disk. The file's
// directory becomes the file context, so COPY sources resolve relative to it.
func FromDockerfile(path string, opts BuilderOptions) (*Builder, error) {
	return New(opts).FromDockerfile(path)
}

// FromDockerfile replaces this builder's contents with a Dockerfile's.
//
// The builder is returned even on failure, so a caller that ignores the error
// gets something usable rather than nil.
func (b *Builder) FromDockerfile(path string) (*Builder, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return b, fmt.Errorf("template: read Dockerfile %q: %w", path, err)
	}

	parsed := New(BuilderOptions{
		FileContextPath: filepath.Dir(path),
		Region:          b.region,
	})
	if err := parsed.parseDockerfile(bytes.NewReader(content)); err != nil {
		return b, fmt.Errorf("template: parse Dockerfile %q: %w", path, err)
	}

	b.baseImage = parsed.baseImage
	b.baseTemplate = parsed.baseTemplate
	b.registry = parsed.registry
	b.startCmd = parsed.startCmd
	b.readyCmd = parsed.readyCmd
	b.instructions = parsed.instructions
	b.fileContextPath = parsed.fileContextPath
	return b, nil
}

// parseDockerfile reads Dockerfile content into the builder.
func (b *Builder) parseDockerfile(r io.Reader) error {
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
			// "FROM base" names the platform's base image rather than an
			// image called "base" on Docker Hub.
			if strings.EqualFold(args[0], "base") {
				b.FromBaseImage()
			} else {
				b.FromImage(args[0])
			}
			seenFrom = true

		case InstructionRun:
			if len(args) == 0 {
				return fmt.Errorf("line %d: RUN requires a command", lineNo)
			}
			b.RunCmd(dockerfileCommand(args))

		case InstructionCopy, InstructionAdd:
			if len(args) < 2 {
				return fmt.Errorf("line %d: %s requires source and destination", lineNo, name)
			}
			// One COPY may name several sources; the build API takes one per
			// step, so they are split out here.
			for _, src := range args[:len(args)-1] {
				instArgs := []string{src, args[len(args)-1]}
				if user := dockerfileChown(node); user != "" {
					instArgs = append(instArgs, user)
				}
				b.addInstruction(Instruction{Type: InstructionCopy, Args: instArgs})
			}

		case InstructionEnv:
			if len(args) < 2 {
				return fmt.Errorf("line %d: invalid ENV instruction", lineNo)
			}
			envs, err := dockerfileEnvArgs(node, args)
			if err != nil {
				return fmt.Errorf("line %d: %w", lineNo, err)
			}
			b.SetEnvs(envs)

		case InstructionArg:
			if len(args) == 0 {
				return fmt.Errorf("line %d: invalid ARG instruction", lineNo)
			}
			for _, arg := range args {
				key, value, hasValue := strings.Cut(arg, "=")
				if key == "" || strings.ContainsAny(key, " \t") {
					return fmt.Errorf("line %d: invalid ARG assignment", lineNo)
				}
				instArgs := []string{key}
				if hasValue {
					instArgs = append(instArgs, value)
				}
				b.addInstruction(Instruction{Type: InstructionArg, Args: instArgs})
			}

		case InstructionWorkdir:
			if len(args) == 0 {
				return fmt.Errorf("line %d: WORKDIR requires a path", lineNo)
			}
			b.SetWorkdir(args[0])

		case InstructionUser:
			if len(args) == 0 {
				return fmt.Errorf("line %d: USER requires a user", lineNo)
			}
			b.SetUser(args[0])

		case InstructionEntrypoint, InstructionCmd:
			if len(args) == 0 {
				return fmt.Errorf("line %d: %s requires a command", lineNo, name)
			}
			b.SetStartCmd(dockerfileCommand(args), ReadyCmd{})

		default:
			log.Printf("template: ignoring unsupported Dockerfile instruction %q on line %d", name, lineNo)
		}
	}

	if !seenFrom {
		return fmt.Errorf("FROM instruction is required")
	}
	return nil
}

// dockerfileNodeArgs flattens a parsed node's argument list.
func dockerfileNodeArgs(node *dockerfileparser.Node) []string {
	var args []string
	for arg := node.Next; arg != nil; arg = arg.Next {
		args = append(args, arg.Value)
	}
	return args
}

// dockerfileEnvArgs reads the key/value pairs out of an ENV or ARG node.
func dockerfileEnvArgs(node *dockerfileparser.Node, args []string) (map[string]string, error) {
	envs := make(map[string]string)

	if strings.EqualFold(node.Value, "ARG") {
		for _, arg := range args {
			key, value, _ := strings.Cut(arg, "=")
			if key == "" || strings.ContainsAny(key, " \t") {
				return nil, fmt.Errorf("invalid ARG assignment")
			}
			envs[key] = value
		}
		return envs, nil
	}

	// BuildKit reports ENV key=value as a key, value, "=" triple, and the
	// older "ENV key value" form as a pair.
	switch {
	case len(args) == 3 && args[2] == "":
		envs[args[0]] = args[1]
		return envs, nil
	case len(args) == 2:
		envs[args[0]] = args[1]
		return envs, nil
	case len(args)%3 != 0:
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

// dockerfileCommand joins a node's arguments back into a shell command.
func dockerfileCommand(args []string) string {
	return strings.Join(args, " ")
}

// dockerfileChown reads the --chown flag off a COPY or ADD node.
func dockerfileChown(node *dockerfileparser.Node) string {
	for _, flag := range node.Flags {
		if strings.HasPrefix(strings.ToLower(flag), "--chown=") {
			return flag[len("--chown="):]
		}
	}
	return ""
}
