// Package mysql exposes the MySQL-family dblog backend.
package mysql

import (
	"context"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mysql/backend"
	"github.com/Infranite/go-dblog/mysql/decode/decoder"
	"github.com/Infranite/go-dblog/mysql/decode/events/types"
)

// Driver is the MySQL-family backend driver name.
const Driver = backend.Driver

// Option configures a MySQL-family binlog decoder.
type Option = decoder.BinFileDecodeOptFunc

// BinFileDecoder streams local MySQL-family binlog files.
type BinFileDecoder = decoder.BinFileDecoder

// DblogDecoder adapts local MySQL-family binlog files to the shared dblog API.
type DblogDecoder = decoder.DblogDecoder

// LiveDecoder streams MySQL replication events.
type LiveDecoder = decoder.LiveDecoder

// EventCompatibilityMode controls how unknown event types are handled.
type EventCompatibilityMode = decoder.EventCompatibilityMode

const (
	// EventCompatibilityAuto checks event type with FORMAT_DESCRIPTION_EVENT metadata.
	EventCompatibilityAuto = decoder.EventCompatibilityAuto
	// EventCompatibilityStrict rejects event types not built into this package.
	EventCompatibilityStrict = decoder.EventCompatibilityStrict
	// EventCompatibilityLoose keeps unknown event types as metadata events.
	EventCompatibilityLoose = decoder.EventCompatibilityLoose
)

// Register adds the MySQL backend to a registry, or to dblog.DefaultRegistry when nil.
func Register(registry *dblog.Registry) error {
	return backend.Register(registry)
}

// NewBinFileDecoder opens a local MySQL-family binlog file.
func NewBinFileDecoder(path string, opts ...Option) (*BinFileDecoder, error) {
	return decoder.NewBinFileDecoder(path, opts...)
}

// NewDblogDecoder opens a local MySQL-family binlog file as a shared dblog decoder.
func NewDblogDecoder(path string, opts ...Option) (*DblogDecoder, error) {
	return decoder.NewDblogDecoder(path, opts...)
}

// NewLiveDecoder opens a MySQL replication stream.
func NewLiveDecoder(ctx context.Context, source dblog.Source, dsn string, opts ...Option) (*LiveDecoder, error) {
	return decoder.NewLiveDecoder(ctx, decoder.Source{Driver: source.Driver, Name: source.Name}, dsn, opts...)
}

// WithStartPos skips local binlog events ending before startPos.
func WithStartPos(startPos int64) Option {
	return decoder.WithStartPos(startPos)
}

// WithEventCompatibilityMode sets unknown-event compatibility behavior.
func WithEventCompatibilityMode(mode EventCompatibilityMode) Option {
	return decoder.WithEventCompatibilityMode(mode)
}

// WithEventPlugins installs event plugins for MySQL-family dialect extensions.
func WithEventPlugins(plugins ...types.EventPlugin) Option {
	return decoder.WithEventPlugins(plugins...)
}
