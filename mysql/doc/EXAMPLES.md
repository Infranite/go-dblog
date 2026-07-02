# MySQL-family Examples

[中文](./EXAMPLES.zh-CN.md)

## Decode A Local Binlog File

```go
package main

import (
	"fmt"

	"github.com/Infranite/go-dblog/mysql/common"
	"github.com/Infranite/go-dblog/mysql/decode/decoder"
)

func main() {
	fileDecoder, err := decoder.NewBinFileDecoder("./testdata/mysql-bin.000004")
	if err != nil {
		panic(err)
	}
	defer fileDecoder.Close()

	for event, err := range fileDecoder.Events() {
		if err != nil {
			panic(err)
		}
		fmt.Println(common.EventTypeName(event.Header.EventType), event.Header.LogPos)
	}
}
```

## Open Through The Common Registry

Use the root registry when a pipeline routes multiple backend types through the
same `dblog.Event` interface.

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

var registry dblog.Registry
if err := mysql.Register(&registry); err != nil {
	panic(err)
}

stream, err := registry.Open(mysql.Driver,
	dblog.WithContext(ctx),
	dblog.WithDSN("mysql://dblog:dblog@127.0.0.1:3306/?server_id=1002"),
)
if err != nil {
	panic(err)
}
defer stream.Close()

for event, err := range stream.Events() {
	if err != nil {
		panic(err)
	}
	fmt.Println(event.Kind(), dblog.PositionOf(event).Value)
}
```

Add `binlog` or `file` plus `pos` query parameters to start from a known binary
log position:

```text
mysql://dblog:dblog@127.0.0.1:3306/?server_id=1002&file=mysql-bin.000004&pos=123
```

## Filter Typed Event Bodies

```go
fileDecoder, err := decoder.NewBinFileDecoder("./testdata/mysql-bin.000004")
if err != nil {
	panic(err)
}
defer fileDecoder.Close()

for queryEvent, err := range decoder.EventBodies[*types.QueryEvent](fileDecoder.Events()) {
	if err != nil {
		panic(err)
	}
	fmt.Println(queryEvent.Schema, queryEvent.Query)
}
```

## Enable A Compatibility Mode

```go
fileDecoder, err := decoder.NewBinFileDecoder(
	"./mysql-bin.000001",
	decoder.WithEventCompatibilityMode(decoder.EventCompatibilityLoose),
)
if err != nil {
	panic(err)
}
defer fileDecoder.Close()
```

## Register A Dialect Plugin

```go
fileDecoder, err := decoder.NewBinFileDecoder(
	"./mysql-bin.000001",
	decoder.WithEventPlugins(myPlugin),
)
if err != nil {
	panic(err)
}
defer fileDecoder.Close()
```

## Build A Recovery Plan

`dblog.RecoveryPlan` emits reverse row events only for complete row images. Each
step carries the checkpoint of the original event; persist it after the reverse
event is durably replayed.

```go
stream, err := decoder.NewDblogDecoder("./testdata/mysql-bin.000004")
if err != nil {
	panic(err)
}
defer stream.Close()

for step, err := range dblog.RecoveryPlan(dblog.Events(stream)) {
	if err != nil {
		panic(err)
	}
	reverse := step.Operation.(*events.Event)
	fmt.Println(step.Checkpoint.Position.Value, reverse.Header.Type())
}
```

For bounded point-in-time rollback, persist the emitted steps and replay them in
reverse checkpoint order.

## Local Checks

From the repository root:

```bash
cd mysql && GOWORK=off go test ./...
make integration-mysql
```
