// Command secret walks a secret through its whole lifecycle, then shows how a
// sandbox reaches its value without ever holding it.
//
//	go run ./examples/secret
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/secret"
)

func main() {
	ctx := context.Background()

	c, err := client.New(client.Options{})
	if err != nil {
		log.Fatal(err)
	}
	secrets := c.Secrets()

	const name = "example-key"

	created, err := secrets.Create(ctx, name, "first-value", secret.CreateOptions{
		Metadata: map[string]string{"example": "secret"},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer secrets.Delete(ctx, created.SecretID)

	fmt.Printf("created %s at version %d\n", created.SecretID, created.Version)

	// A secret can be named by ID or by name.
	got, err := secrets.GetInfo(ctx, name)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("metadata: %v\n", got.Metadata)

	updated, err := secrets.Update(ctx, name, "second-value", secret.UpdateOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("updated to version %d\n", updated.Version)

	all, err := secrets.List(ctx, secret.ListOptions{}).All(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("project has %d secret(s)\n", len(all))

	// The sandbox is given a placeholder, not the value. The runtime resolves
	// it on the way out, so the value is never in the sandbox's environment.
	sbx, err := c.Sandboxes().Create(ctx, sandbox.CreateOptions{
		Template: "base",
		EnvVars:  map[string]string{"EXAMPLE_KEY": secret.MustFill(name)},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer sbx.Kill(ctx)

	out, err := sbx.Commands.Run(ctx, "printenv EXAMPLE_KEY", sandbox.CommandOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("as seen inside the sandbox: ", out.Stdout)
}
