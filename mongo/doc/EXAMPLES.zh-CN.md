# MongoDB-family 示例

[English](./EXAMPLES.md)

## 解析 Oplog JSON Record

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

## 解析 Change Stream Capture

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

source name 使用 `db.collection` 形式。MongoDB 必须以 replica set 运行。

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

如果需要从 live change stream 生成 update 闪回，需要在 collection 上启用 pre-images。

## 遍历闪回命令

```go
for event, err := range dblog.Flashbacks(decoder.Events()) {
	if err != nil {
		panic(err)
	}
	command := event.Body().(mongo.Command)
	fmt.Println(command.Operation, command.Filter, command.Document)
}
```

## 注册事件插件

```go
package main

import (
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo/decode/decoder"
	"github.com/Infranite/go-dblog/mongo/decode/events/types"
)

type replacePlugin struct{}

func (replacePlugin) Name() string { return "replace" }
func (replacePlugin) Match(raw map[string]any) bool {
	return raw["operationType"] == "replace"
}
func (replacePlugin) Apply(change *types.Change) error {
	change.Operation = types.OperationUpdate
	return nil
}

func main() {
	_ = decoder.NewDecoder(
		dblog.Source{Name: "changes"},
		strings.NewReader(`{"operationType":"replace","ns":{"db":"app","coll":"users"}}`+"\n"),
		nil,
		decoder.WithEventPlugins(replacePlugin{}),
	)
}
```

## 本地检查

在仓库根目录执行：

```bash
cd mongo && GOWORK=off go test ./...
make integration-mongo
```
