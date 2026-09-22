// Command template builds a template and starts a sandbox from it.
//
//	go run ./examples/template
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/template"
)

func main() {
	ctx := context.Background()

	c, err := client.New(client.Options{})
	if err != nil {
		log.Fatal(err)
	}
	tpls := c.Templates()

	// NewBuilder takes the region from the client, so the default base image
	// is the one reachable from there.
	spec := tpls.NewBuilder(template.BuilderOptions{}).
		FromImage("ubuntu:22.04").
		RunCmd("apt-get update && apt-get install -y python3").
		SetWorkdir("/app")

	build, err := tpls.Build(ctx, spec, "example-python", template.BuildOptions{
		CPUCount: 2,
		MemoryMB: 2048,
		OnLogs:   template.DefaultLogger(),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("built template:", build.TemplateID)

	sbx, err := c.Sandboxes().Create(ctx, sandbox.CreateOptions{Template: build.TemplateID})
	if err != nil {
		log.Fatal(err)
	}
	defer sbx.Kill(ctx)

	out, err := sbx.Commands.Run(ctx, "python3 --version", sandbox.CommandOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(out.Stdout)
}
