# PostgreSQL-family 示例

[English](./EXAMPLES.md)

## 解析 Logical Decoding 文本

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres"
)

func main() {
	var registry dblog.Registry
	if err := postgres.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(postgres.Driver,
		dblog.WithSource(dblog.Source{Name: "slot"}),
		dblog.WithReader(strings.NewReader("table public.users: INSERT: id[integer]:1 name[text]:'Ada'\n")),
	)
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	for event, err := range decoder.Events() {
		if err != nil {
			panic(err)
		}
		change := event.Body().(postgres.Change)
		fmt.Println(event.Kind(), change.Schema, change.Table, len(change.Columns))
	}
}
```

## Live SQL Slot Reader

注册 backend，并用 PostgreSQL DSN、context 以及匹配 logical slot 的 source name 打开：

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

decoder, err := registry.Open(postgres.Driver,
	dblog.WithContext(ctx),
	dblog.WithDSN("postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"),
	dblog.WithSource(dblog.Source{Name: "dblog_slot"}),
)
if err != nil {
	panic(err)
}
defer decoder.Close()
```

## Wire-level Logical Replication

DSN 增加 `replication=database` 时使用 wire-level replication：

```go
decoder, err := registry.Open(postgres.Driver,
	dblog.WithContext(ctx),
	dblog.WithDSN("postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable&replication=database"),
	dblog.WithSource(dblog.Source{Name: "dblog_slot"}),
)
if err != nil {
	panic(err)
}
defer decoder.Close()
```

## 构建带 Checkpoint 的恢复计划

```go
for step, err := range dblog.RecoveryPlan(decoder.Events()) {
	if err != nil {
		panic(err)
	}
	sql := step.Operation.(string)
	fmt.Println(step.Checkpoint.Position.Value, sql)
}
```

Update 闪回需要完整 old tuple 和 new tuple 数据。需要 PostgreSQL 为 update 输出完整
old tuple 时，使用 `REPLICA IDENTITY FULL`。反向 SQL 持久执行成功后再保存 checkpoint。

## 注册文本事件插件

```go
package main

import (
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/decoder"
	"github.com/Infranite/go-dblog/postgres/decode/events/types"
)

type messagePlugin struct{}

func (messagePlugin) Name() string { return "message" }
func (messagePlugin) Match(line string) bool {
	return strings.HasPrefix(line, "message ")
}
func (messagePlugin) Decode(source dblog.Source, position int, line string) (types.Event, error) {
	return types.NewEvent(source, position, []byte(line), "message", line), nil
}

func main() {
	_ = decoder.NewDecoder(
		dblog.Source{Name: "slot"},
		strings.NewReader("message hello\n"),
		nil,
		decoder.WithEventPlugins(messagePlugin{}),
	)
}
```

## 本地检查

在仓库根目录执行：

```bash
cd postgres && GOWORK=off go test ./...
make integration-postgres
```
