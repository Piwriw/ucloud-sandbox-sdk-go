// Command volume creates a volume and mounts it into a sandbox.
//
//	go run ./examples/volume
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
)

func main() {
	ctx := context.Background()

	c, err := client.New(client.Options{})
	if err != nil {
		log.Fatal(err)
	}

	vol, err := c.Volumes().Create(ctx, "example-data")
	if err != nil {
		log.Fatal(err)
	}
	defer vol.Destroy(ctx)

	fmt.Println("volume:", vol.ID)

	sbx, err := c.Sandboxes().Create(ctx, sandbox.CreateOptions{
		Template:     "base",
		VolumeMounts: []api.SandboxVolumeMount{vol.Mount("/mnt/data")},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer sbx.Kill(ctx)

	// The volume is written from inside the sandbox, where it is mounted.
	out, err := sbx.Commands.Run(ctx,
		"echo persisted > /mnt/data/note.txt && cat /mnt/data/note.txt",
		sandbox.CommandOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(out.Stdout)
}
