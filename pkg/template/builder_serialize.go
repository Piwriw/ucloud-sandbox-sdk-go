package template

import (
	"encoding/json"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

// buildableSteps are the instruction types the build API accepts. A Builder can
// hold others — ENTRYPOINT and CMD arrive from a Dockerfile — which are carried
// as the start command instead of as steps, so sending them would be rejected.
var buildableSteps = map[InstructionType]bool{
	InstructionCopy:    true,
	InstructionAdd:     true,
	InstructionRun:     true,
	InstructionUser:    true,
	InstructionWorkdir: true,
	InstructionEnv:     true,
	InstructionArg:     true,
}

// steps returns the instructions the build API will accept.
func (b *Builder) steps() []Instruction {
	steps := make([]Instruction, 0, len(b.instructions))
	for _, inst := range b.instructions {
		if buildableSteps[inst.Type] {
			steps = append(steps, inst)
		}
	}
	return steps
}

// serialize renders the builder into a build-start request body.
//
// steps is passed in rather than read off the builder because Build computes
// the file hashes first; they are part of each COPY step's cache key.
func (b *Builder) serialize(steps []Instruction, force bool) (api.TemplateBuildStartV2, error) {
	body := api.TemplateBuildStartV2{}

	if b.baseImage != "" {
		body.FromImage = &b.baseImage
	}
	if b.baseTemplate != "" {
		body.FromTemplate = &b.baseTemplate
	}
	if b.startCmd != "" {
		body.StartCmd = &b.startCmd
	}
	if b.readyCmd != "" {
		body.ReadyCmd = &b.readyCmd
	}

	if force || b.force {
		forced := true
		body.Force = &forced
	}

	registry, err := b.registry.apiValue()
	if err != nil {
		return api.TemplateBuildStartV2{}, err
	}
	body.FromImageRegistry = registry

	converted := make([]api.TemplateStep, 0, len(steps))
	for _, step := range steps {
		converted = append(converted, apiStep(step))
	}
	body.Steps = &converted

	return body, nil
}

// apiStep converts one instruction into the generated step type.
func apiStep(step Instruction) api.TemplateStep {
	converted := api.TemplateStep{Type: string(step.Type)}
	if len(step.Args) > 0 {
		args := step.Args
		converted.Args = &args
	}
	if step.Force {
		force := true
		converted.Force = &force
	}
	if step.FilesHash != "" {
		hash := step.FilesHash
		converted.FilesHash = &hash
	}
	return converted
}

// ToJSON renders the template description the build API would receive.
//
// COPY steps are rendered without their file hashes, which are only computed
// during a build.
func (b *Builder) ToJSON() (string, error) {
	if b.err != nil {
		return "", b.err
	}
	body, err := b.serialize(b.steps(), false)
	if err != nil {
		return "", err
	}
	encoded, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
