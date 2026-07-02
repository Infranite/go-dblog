# Redis-family Examples

[中文](./EXAMPLES.zh-CN.md)

## Decode AOF RESP Commands

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

The live reader consumes the initial PSYNC RDB snapshot payload and starts
emitting following RESP command frames.

## Build A Recovery Plan With Checkpoints

```go
for step, err := range dblog.RecoveryPlan(decoder.Events()) {
	if err != nil {
		panic(err)
	}
	command := step.Operation.(redis.Command)
	fmt.Println(step.Checkpoint.Position.Value, command.Name, command.Args)
}
```

Only commands with deterministic reverse commands are emitted. This includes
list pushes, counter increments, `HINCRBY`, `HINCRBYFLOAT`, and `ZINCRBY`.
State-dependent commands such as `SET`, `HSET`, `SADD`, and `DEL` are omitted.
Persist the checkpoint after the compensating command is durably applied.

## Register A Command Plugin

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

## Local Checks

From the repository root:

```bash
cd redis && GOWORK=off go test ./...
make integration-redis
```
