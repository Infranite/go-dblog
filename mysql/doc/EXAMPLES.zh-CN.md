# MySQL-family 示例

[English](./EXAMPLES.md)

## 解析本地 Binlog 文件

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

## 通过公共 Registry 打开

多 backend pipeline 需要统一使用 `dblog.Event` 时，使用根 registry。

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

通过 `binlog` 或 `file` 加 `pos` query 参数可以从指定 binary log position 开始：

```text
mysql://dblog:dblog@127.0.0.1:3306/?server_id=1002&file=mysql-bin.000004&pos=123
```

## 过滤 Typed Event Body

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

## 启用兼容模式

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

## 注册方言插件

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

## 遍历闪回事件

`dblog.Flashbacks` 只为完整 row image 输出反向 row event。

```go
events := dblog.Flashbacks(stream.Events())
for event, err := range events {
	if err != nil {
		panic(err)
	}
	fmt.Println(event.Kind(), dblog.PositionOf(event).Value)
}
```

## 本地检查

在仓库根目录执行：

```bash
cd mysql && GOWORK=off go test ./...
make integration-mysql
```
