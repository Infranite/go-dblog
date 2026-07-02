# MongoDB-family Examples

[中文](./EXAMPLES.zh-CN.md)

## Decode An Oplog JSON Record

```go
package main

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo"
)

func main() {
	var registry dblog.Registry
	if err := mongo.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(mongo.Driver,
		dblog.WithReader(strings.NewReader(`{"op":"i","ns":"app.users","o":{"_id":1,"name":"Ada"}}`+"\n")),
	)
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	for event, err := range decoder.Events() {
		if err != nil {
			panic(err)
		}
		change := event.Body().(mongo.Change)
		fmt.Println(event.Kind(), change.Database, change.Collection)
	}
}
```

## Decode A Change Stream Capture

```go
input := strings.NewReader(
	`{"operationType":"insert","ns":{"db":"app","coll":"users"},"documentKey":{"_id":1},"fullDocument":{"_id":1,"name":"Ada"}}` + "\n",
)

decoder, err := registry.Open(mongo.Driver, dblog.WithReader(input))
if err != nil {
	panic(err)
}
defer decoder.Close()
```

## Live Change Stream Reader

Use `db.collection` as the source name. MongoDB must run as a replica set.

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

decoder, err := registry.Open(mongo.Driver,
	dblog.WithContext(ctx),
	dblog.WithDSN("mongodb://127.0.0.1:27017"),
	dblog.WithSource(dblog.Source{Name: "app.users"}),
)
if err != nil {
	panic(err)
}
defer decoder.Close()
```

Enable collection pre-images when update or replace flashbacks must be emitted
from live change streams.

## Build A Recovery Plan With Pre-Images

```go
for step, err := range dblog.RecoveryPlan(decoder.Events()) {
	if err != nil {
		panic(err)
	}
	command := step.Operation.(mongo.Command)
	fmt.Println(step.Checkpoint.Position.Value, command.Operation, command.Filter, command.Document)
}
```

For live update or replace recovery, enable collection pre-images before opening
the stream:

```javascript
db.runCommand({
  collMod: "users",
  changeStreamPreAndPostImages: { enabled: true }
})
```

Without `fullDocumentBeforeChange`, update and replace events are decoded but no
recovery step is emitted.

## Register An Event Plugin

```go
package main

import (
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo/decode/decoder"
	"github.com/Infranite/go-dblog/mongo/decode/events/types"
)

type upsertPlugin struct{}

func (upsertPlugin) Name() string { return "upsert" }
func (upsertPlugin) Match(raw map[string]any) bool {
	return raw["operationType"] == "upsert"
}
func (upsertPlugin) Apply(change *types.Change) error {
	change.Operation = types.OperationUpdate
	return nil
}

func main() {
		_ = decoder.NewDecoder(
		dblog.Source{Name: "changes"},
		strings.NewReader(`{"operationType":"upsert","ns":{"db":"app","coll":"users"}}`+"\n"),
		nil,
		decoder.WithEventPlugins(upsertPlugin{}),
	)
}
```

## Local Checks

From the repository root:

```bash
cd mongo && GOWORK=off go test ./...
make integration-mongo
```
