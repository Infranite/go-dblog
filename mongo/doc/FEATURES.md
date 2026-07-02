# MongoDB-family Features And Scope

[中文](./FEATURES.zh-CN.md)

This document is the detailed feature and support reference for the
MongoDB-family backend.

## Packages

| Package | Purpose |
|---|---|
| `github.com/Infranite/go-dblog/mongo` | Compatibility facade for common imports. |
| `github.com/Infranite/go-dblog/mongo/backend` | Explicit registration with `dblog.Registry`. |
| `github.com/Infranite/go-dblog/mongo/decode/decoder` | Native streaming decoder, line parser, and plugin options. |
| `github.com/Infranite/go-dblog/mongo/decode/events/types` | Native change, command, event, and plugin types. |

## Supported

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

## Unsupported

- Raw oplog tailing outside JSON records or change streams.
- Automatic replica set or sharded cluster discovery.
- Update flashback without `fullDocumentBeforeChange`.
- Delete flashback without full deleted document data.

## Supported Inputs

| Input | Status | CI evidence |
|---|---|---|
| MongoDB oplog JSON records with `op`, `ns`, `o`, and `o2` | Supported | `mongo` fixture job generated from `mongo:7.0`; `FuzzParseLine` smoke target. |
| MongoDB change stream JSON records with `operationType`, `ns`, `documentKey`, `fullDocument`, `fullDocumentBeforeChange`, and `updateDescription` | Supported | Unit tests and `FuzzParseLine` seeds cover valid and malformed records. |
| Live collection change streams from MongoDB replica sets | Supported | `mongo` CI job starts `mongo:7.0`, opens a live stream, writes insert/update/delete operations, and reads them through `dblog.WithDSN` plus `dblog.WithContext`. |
| Malformed JSON records or non-object `updateDescription` values | Rejected | `TestParseLineRejectsMalformedInput` and `FuzzParseLine`. |
| Empty operation names | Rejected | Parser tests and fuzz smoke target. |
| Unknown non-empty operation names | Emitted as backend event kinds unless a decoder plugin normalizes them | Plugin tests and parser tests. |

## Live Change Streams

Open a live reader with `dblog.WithDSN` and a source name in `db.collection`
form. MongoDB must be running as a replica set because standalone servers do not
support change streams.

Update flashback for live change streams requires `fullDocumentBeforeChange`.
Enable change stream pre-images on the source collection when reverse update
commands are required. Without a pre-image, update events are still decoded, but
`dblog.Flashbacks` intentionally emits no reverse command.

## Flashback Scope

| Event | Flashback output |
|---|---|
| `insert` with `documentKey` | `mongo.Command{Operation: "delete", Filter: documentKey}` |
| `update` with `documentKey` and `fullDocumentBeforeChange` | `mongo.Command{Operation: "replace", Filter: documentKey, Document: fullDocumentBeforeChange}` |
| `delete` with full document data | `mongo.Command{Operation: "insert", Document: document}` |
| `update` without before-image, `command`, `noop` | No flashback output. |

Update flashback uses the full before-image as a replacement document. Updates
without before-image data do not emit flashback output. Malformed JSON input and
non-object `updateDescription` values are rejected before an event is emitted.

## Plugin Support

Use `decoder.WithEventPlugins` when a MongoDB-compatible source emits an event
shape this module should normalize before exposing it to callers. Plugins can
rename operations, fill product-specific metadata, or map compatible event
families into the backend-native `types.Change` shape.
