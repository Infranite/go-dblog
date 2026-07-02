# Redis-family 示例

[English](./EXAMPLES.md)

## 解析 AOF RESP Commands

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis"
)

func main() {
	var registry dblog.Registry
	if err := redis.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(redis.Driver,
		dblog.WithSource(dblog.Source{Name: "appendonly.aof"}),
		dblog.WithReader(strings.NewReader("*3\r\n$4\r\nSADD\r\n$4\r\ntags\r\n$2\r\ngo\r\n")),
	)
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	for event, err := range decoder.Events() {
		if err != nil {
			panic(err)
		}
		command := event.Body().(redis.Command)
		fmt.Println(event.Kind(), command.Args)
	}
}
```

## Live Replication Reader

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

decoder, err := registry.Open(redis.Driver,
	dblog.WithContext(ctx),
	dblog.WithDSN("redis://127.0.0.1:6379"),
	dblog.WithSource(dblog.Source{Name: "primary"}),
)
if err != nil {
	panic(err)
}
defer decoder.Close()
```

live reader 会消费初始 PSYNC RDB snapshot payload，然后从后续 RESP command frames
开始输出事件。

## 构建带 Checkpoint 的恢复计划

```go
for step, err := range dblog.RecoveryPlan(decoder.Events()) {
	if err != nil {
		panic(err)
	}
	command := step.Operation.(redis.Command)
	fmt.Println(step.Checkpoint.Position.Value, command.Name, command.Args)
}
```

只有具备确定性反向命令的操作会输出闪回，包括 list push、counter increment、
`HINCRBY`、`HINCRBYFLOAT` 和 `ZINCRBY`。`SET`、`HSET`、`SADD`、`DEL` 等依赖状态的
命令会被省略。补偿命令持久执行成功后再保存 checkpoint。

## 注册命令插件

```go
package main

import (
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis/decode/decoder"
	"github.com/Infranite/go-dblog/redis/decode/events/types"
)

type jsonSetPlugin struct{}

func (jsonSetPlugin) Name() string { return "json-set" }
func (jsonSetPlugin) Match(command types.Command) bool {
	return command.Name == "json.set"
}
func (jsonSetPlugin) Apply(command *types.Command) error {
	command.Name = "jsonset"
	return nil
}

func main() {
	_ = decoder.NewDecoder(
		dblog.Source{Name: "appendonly.aof"},
		strings.NewReader("*2\r\n$8\r\nJSON.SET\r\n$5\r\nkey:1\r\n"),
		nil,
		decoder.WithCommandPlugins(jsonSetPlugin{}),
	)
}
```

## 本地检查

在仓库根目录执行：

```bash
cd redis && GOWORK=off go test ./...
make integration-redis
```
