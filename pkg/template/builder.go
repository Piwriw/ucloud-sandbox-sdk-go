package template

import (
	"fmt"
	"sort"
)

// InstructionType is the kind of a build step. The names match Dockerfile
// instructions, which is what the platform's build steps are modelled on.
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

// Instruction is one build step.
type Instruction struct {
	Type InstructionType
	Args []string

	// Force rebuilds this step even when the cache holds a result for it.
	Force bool

	// FilesHash identifies the uploaded file bundle a COPY step applies. It is
	// computed during Build; callers do not set it.
	FilesHash string
}

// ReadyCmd is a shell command the platform runs until it succeeds, to decide
// that a started template is ready to serve. Build one with WaitForPort,
// WaitForFile or WaitForTimeout, or write the command yourself.
type ReadyCmd struct {
	Cmd string
}

func (r ReadyCmd) String() string { return r.Cmd }

// WaitForPort waits until something is listening on port.
func WaitForPort(port int) ReadyCmd {
	return ReadyCmd{Cmd: fmt.Sprintf("ss -tuln | grep :%d", port)}
}

// WaitForFile waits until filename exists.
func WaitForFile(filename string) ReadyCmd {
	return ReadyCmd{Cmd: fmt.Sprintf("[ -f %s ]", filename)}
}

// WaitForTimeout waits for a fixed duration, at least one second.
func WaitForTimeout(timeoutMs int) ReadyCmd {
	if timeoutMs < 1000 {
		timeoutMs = 1000
	}
	return ReadyCmd{Cmd: fmt.Sprintf("sleep %.3f", float64(timeoutMs)/1000.0)}
}

// BuilderOptions are the optional arguments to New and Service.NewBuilder.
type BuilderOptions struct {
	// FileContextPath is the local directory COPY sources are resolved
	// against. Required before a template with COPY steps can be built.
	FileContextPath string

	// Region selects the default base image. Ignored when BaseImage is set.
	// Service.NewBuilder fills this in from the client's configuration;
	// New falls back to UCLOUD_SANDBOX_REGION and then to the default region.
	Region string

	// BaseImage overrides the default base image outright.
	BaseImage string
}

// Builder describes a template. Its methods chain:
//
//	tpl := tpls.NewBuilder(template.BuilderOptions{}).
//	    FromImage("ubuntu:22.04").
//	    RunCmd("apt-get update")
//
// Errors are recorded rather than returned, so a chain reads cleanly; the first
// one surfaces from Build, ToJSON or ToDockerfile.
type Builder struct {
	// region picks the default base image; see BaseImageForRegion.
	region string

	baseImage    string
	baseTemplate string
	registry     *Registry

	startCmd string
	readyCmd string

	force          bool
	forceNextLayer bool

	instructions    []Instruction
	fileContextPath string

	err error
}

// New returns a Builder that is not tied to a client.
//
// Prefer Service.NewBuilder, which knows the client's region and so picks the
// right default base image. Use this when building a template description
// without a client to hand — rendering a Dockerfile, say.
func New(opts BuilderOptions) *Builder {
	region := opts.Region
	if region == "" {
		region = defaultRegion()
	}

	baseImage := opts.BaseImage
	if baseImage == "" {
		baseImage = BaseImageForRegion(region)
	}

	return &Builder{
		region:          region,
		baseImage:       baseImage,
		fileContextPath: opts.FileContextPath,
	}
}

// Err returns the first error recorded while building the chain.
func (b *Builder) Err() error { return b.err }

// FromImage starts the template from a public container image.
func (b *Builder) FromImage(image string) *Builder {
	return b.FromImageWithAuth(image, nil)
}

// FromImageWithAuth starts the template from an image in a private registry.
// Build auth with BasicAuth, AWSAuth or GCPAuth.
func (b *Builder) FromImageWithAuth(image string, auth *Registry) *Builder {
	b.baseImage = image
	b.baseTemplate = ""
	b.registry = auth
	if b.forceNextLayer {
		b.force = true
	}
	return b
}

// From is an alias for FromImage.
func (b *Builder) From(image string) *Builder {
	return b.FromImage(image)
}

// FromTemplate starts the template from another template rather than an image.
func (b *Builder) FromTemplate(template string) *Builder {
	b.baseTemplate = template
	b.baseImage = ""
	b.registry = nil
	if b.forceNextLayer {
		b.force = true
	}
	return b
}

// FromBaseImage starts the template from the platform's base image for the
// builder's region. This is already the default; call it to return to that base
// after another From call.
func (b *Builder) FromBaseImage() *Builder {
	return b.FromImage(b.defaultBaseImage())
}

// defaultBaseImage is the platform base image for this builder's region.
func (b *Builder) defaultBaseImage() string {
	return BaseImageForRegion(b.region)
}

func (b *Builder) addInstruction(inst Instruction) {
	inst.Force = b.forceNextLayer
	b.instructions = append(b.instructions, inst)
	b.forceNextLayer = false
}

// RunCmd runs a shell command during the build.
func (b *Builder) RunCmd(command string) *Builder {
	b.addInstruction(Instruction{Type: InstructionRun, Args: []string{command}})
	return b
}

// RunCmdAsUser runs a shell command during the build as a given user.
func (b *Builder) RunCmdAsUser(command, user string) *Builder {
	args := []string{command}
	if user != "" {
		args = append(args, user)
	}
	b.addInstruction(Instruction{Type: InstructionRun, Args: args})
	return b
}

// SetEnvs sets environment variables for the remaining steps and for sandboxes
// started from the template.
//
// The keys are sorted, so the same map always produces the same step and
// therefore the same cache key.
func (b *Builder) SetEnvs(envs map[string]string) *Builder {
	if len(envs) == 0 {
		return b
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
	b.addInstruction(Instruction{Type: InstructionEnv, Args: args})
	return b
}

// SetWorkdir sets the working directory for the remaining steps.
func (b *Builder) SetWorkdir(workdir string) *Builder {
	b.addInstruction(Instruction{Type: InstructionWorkdir, Args: []string{workdir}})
	return b
}

// SetUser sets the user for the remaining steps.
func (b *Builder) SetUser(user string) *Builder {
	b.addInstruction(Instruction{Type: InstructionUser, Args: []string{user}})
	return b
}

// SkipCache forces the next step to run even if the cache holds a result for
// it.
func (b *Builder) SkipCache() *Builder {
	b.forceNextLayer = true
	return b
}

// Copy copies files from the local file context into the template.
//
// src is relative to BuilderOptions.FileContextPath and must not escape it.
// The files are hashed and uploaded by Build before the build starts.
func (b *Builder) Copy(src, dest string) *Builder {
	b.addInstruction(Instruction{Type: InstructionCopy, Args: []string{src, dest}})
	return b
}

// SetStartCmd sets the command a sandbox runs on start, and the check that
// decides when it is ready.
func (b *Builder) SetStartCmd(startCmd string, readyCmd ReadyCmd) *Builder {
	b.startCmd = startCmd
	b.readyCmd = readyCmd.Cmd
	return b
}
