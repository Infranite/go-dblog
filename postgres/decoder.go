package postgres

import (
	"context"
	"io"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/decoder"
	"github.com/Infranite/go-dblog/postgres/decode/events/types"
)

// Option configures a PostgreSQL decoder.
type Option = decoder.Option

// Decoder streams PostgreSQL logical decoding text lines.
type Decoder = decoder.Decoder

// LiveDecoder streams PostgreSQL logical decoding rows from a SQL slot.
type LiveDecoder = decoder.LiveDecoder

// WireLiveDecoder streams PostgreSQL logical decoding rows over replication protocol.
type WireLiveDecoder = decoder.WireLiveDecoder

// NewDecoder creates a decoder over PostgreSQL logical decoding text.
func NewDecoder(source dblog.Source, reader io.Reader, close func() error, opts ...Option) *Decoder {
	return decoder.NewDecoder(source, reader, close, opts...)
}

// NewLiveDecoder creates a decoder over a live PostgreSQL logical decoding slot.
func NewLiveDecoder(ctx context.Context, source dblog.Source, dsn, slot string, opts ...Option) (*decoder.LiveDecoder, error) {
	return decoder.NewLiveDecoder(ctx, source, dsn, slot, opts...)
}

// NewWireLiveDecoder creates a decoder over PostgreSQL logical replication protocol.
func NewWireLiveDecoder(ctx context.Context, source dblog.Source, dsn, slot string, opts ...Option) (*decoder.WireLiveDecoder, error) {
	return decoder.NewWireLiveDecoder(ctx, source, dsn, slot, opts...)
}

// WithEventPlugins installs event plugins for PostgreSQL logical decoding extensions.
func WithEventPlugins(plugins ...types.EventPlugin) Option {
	return decoder.WithEventPlugins(plugins...)
}
