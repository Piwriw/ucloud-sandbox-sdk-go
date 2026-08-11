# ucloud-sandbox-sdk-go

UCloud Sandbox 的 Go SDK，提供与 [E2B](https://e2b.dev) 兼容的沙箱控制面 API 封装：创建/连接沙箱、执行命令、文件操作、持久化 Volume、快照与模板管理等。

仓库地址：[github.com/ucloud/ucloud-sandbox-sdk-go](https://github.com/ucloud/ucloud-sandbox-sdk-go)

## 安装

```bash
go get github.com/ucloud/ucloud-sandbox-sdk-go
```

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ucloud/ucloud-sandbox-sdk-go"
)

func main() {
	ctx := context.Background()
	client := sandbox.NewClient("cn-wlcb.sandbox.ucloudai.com", "your-api-key")

	sbx, err := client.CreateSandbox(ctx, sandbox.WithTemplate("base"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.KillSandbox(ctx, sbx.ID)

	out, err := sbx.Commands.Run(ctx, "uname -a")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out.Stdout)
}
```
