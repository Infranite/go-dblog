# MongoDB-family backend

[![Go Reference](https://pkg.go.dev/badge/github.com/Infranite/go-dblog/mongo.svg)](https://pkg.go.dev/github.com/Infranite/go-dblog/mongo)

This module is the MongoDB-family backend for `go-dblog`. It decodes newline
delimited JSON records from MongoDB oplog exports or change stream captures and
keeps MongoDB-specific fields in typed events. It can also stream live
collection change events from a MongoDB replica set.

[中文](./README.zh-CN.md)

Use the root [`go-dblog`](../README.md) module when you need multi-source
orchestration. Use this module directly when you only need MongoDB-family log
parsing.

## Installation

No public tags have been published yet. After the first `v0.1.0` tag set is
published:

```bash
go get github.com/Infranite/go-dblog/mongo@v0.1.0
```

The repository tag for this module is `mongo/v0.1.0`; callers use the semantic
version query above with `go get`.

Requirements:

- Go 1.25 or later.
- One JSON record per line from an oplog export or change stream capture.
- A MongoDB replica set when opening a live change stream through a DSN.

## Quick Start

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

## Packages

| Package | Purpose |
|---|---|
| `github.com/Infranite/go-dblog/mongo` | Compatibility facade for common imports. |
| `github.com/Infranite/go-dblog/mongo/backend` | Explicit registration with `dblog.Registry`. |
| `github.com/Infranite/go-dblog/mongo/decode/decoder` | Native streaming decoder, line parser, and plugin options. |
| `github.com/Infranite/go-dblog/mongo/decode/events/types` | Native change, command, event, and plugin types. |

## Features

- Oplog JSON records with `op`, `ns`, `o`, and `o2` fields.
- Change stream JSON records with `operationType`, `ns`, `documentKey`,
  `fullDocument`, `fullDocumentBeforeChange`, and `updateDescription`.
- Live collection change streams opened with `dblog.WithDSN` and
  `dblog.WithSource(dblog.Source{Name: "db.collection"})`.
- Streaming line decoder with bounded scanner buffers.
- Root registry integration through `mongo/backend`.
- Checkpoint resume through `dblog.WithCheckpoint` when opened through the root
  registry.
- Flashback commands for inserts, deletes, and updates when the input contains
  enough document or key data.
- Event plugins for MongoDB-compatible products that emit different operation
  names or metadata.

## Supported Inputs

| Input | Status | CI evidence |
|---|---|---|
| MongoDB oplog JSON records with `op`, `ns`, `o`, and `o2` | Supported | `mongo` fixture job generated from `mongo:7.0`; `FuzzParseLine` smoke target. |
| MongoDB change stream JSON records with `operationType`, `ns`, `documentKey`, `fullDocument`, `fullDocumentBeforeChange`, and `updateDescription` | Supported | Unit tests and `FuzzParseLine` seeds cover valid and malformed records. |
| Live collection change streams from MongoDB replica sets | Supported | `mongo` CI job starts `mongo:7.0`, opens a live stream, writes insert/update/delete operations, and reads them through `dblog.WithDSN` plus `dblog.WithContext`. |
| Empty operation names | Rejected | Parser tests and fuzz smoke target. |
| Unknown non-empty operation names | Emitted as backend event kinds unless a decoder plugin normalizes them | Plugin tests and parser tests. |

## Flashback Scope

| Event | Flashback output |
|---|---|
| `insert` with `documentKey` | `mongo.Command{Operation: "delete", Filter: documentKey}` |
| `update` with `documentKey` and `fullDocumentBeforeChange` | `mongo.Command{Operation: "replace", Filter: documentKey, Document: fullDocumentBeforeChange}` |
| `delete` with full document data | `mongo.Command{Operation: "insert", Document: document}` |
| `update` without before-image, `command`, `noop` | No flashback output. |

Update flashback uses the full before-image as a replacement document. Updates
without before-image data do not emit flashback output.

## Event Plugins

Use `decoder.WithEventPlugins` when a MongoDB-compatible source emits an event
shape this module should normalize before exposing it to callers.

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

## Development

From the repository root, run:

```bash
cd mongo && GOWORK=off go test ./...
```

Run the MongoDB fixture-backed integration test locally when Docker is
available:

```bash
make integration-mongo
```

## License

Apache License 2.0. See [LICENSE](../LICENSE).
