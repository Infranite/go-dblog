# PostgreSQL-family Examples

[中文](./EXAMPLES.zh-CN.md)

## Decode Logical Decoding Text

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

Register the backend and open it with a PostgreSQL DSN, a context, and a source
name matching the logical slot.

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

Use wire-level replication by adding `replication=database` to the DSN:

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

## Build A Recovery Plan With Checkpoints

```go
for step, err := range dblog.RecoveryPlan(decoder.Events()) {
	if err != nil {
		panic(err)
	}
	sql := step.Operation.(string)
	fmt.Println(step.Checkpoint.Position.Value, sql)
}
```

Update flashback needs complete old and new tuple data. Use `REPLICA IDENTITY
FULL` when PostgreSQL must emit a complete old tuple for updates. Persist the
checkpoint after the reverse SQL is durably applied.

## Register A Text Event Plugin

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

## Local Checks

From the repository root:

```bash
cd postgres && GOWORK=off go test ./...
make integration-postgres
```
