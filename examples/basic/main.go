// Command basic creates a sandbox, runs a command in it, writes and reads a
// file, then shuts it down.
//
// Set UCLOUD_SANDBOX_API_KEY before running:
//
//	go run ./examples/basic
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/sandbox"
)

func main() {
	ctx := context.Background()

	c, err := client.New(client.Options{})
	if err != nil {
		log.Fatal(err)
	}

	sbx, err := c.Sandboxes().Create(ctx, sandbox.CreateOptions{
		Template:       "base",
		TimeoutSeconds: 300,
		Metadata:       map[string]string{"example": "basic"},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if _, err := sbx.Kill(ctx); err != nil {
			log.Printf("kill sandbox: %v", err)
		}
	}()

	fmt.Println("sandbox:", sbx.ID)

	out, err := sbx.Commands.Run(ctx, "uname -a", sandbox.CommandOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(out.Stdout)

	if _, err := sbx.Files.Write(ctx, "/home/user/hello.txt", "hello\n", sandbox.FileOptions{}); err != nil {
		log.Fatal(err)
	}

	content, err := sbx.Files.Read(ctx, "/home/user/hello.txt", sandbox.FileOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("read back: ", content)

	entries, err := sbx.Files.List(ctx, "/home/user", sandbox.FileOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range entries {
		fmt.Printf("%s %s\n", entry.Type, entry.Path)
	}
}
