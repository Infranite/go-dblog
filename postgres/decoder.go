package postgres

import (
	"io"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/decoder"
	"github.com/Infranite/go-dblog/postgres/decode/events/types"
)

// Option configures a PostgreSQL decoder.
type Option = decoder.Option

// Decoder streams PostgreSQL logical decoding text lines.
type Decoder = decoder.Decoder

// NewDecoder creates a decoder over PostgreSQL logical decoding text.
func NewDecoder(source dblog.Source, reader io.Reader, close func() error, opts ...Option) *Decoder {
	return decoder.NewDecoder(source, reader, close, opts...)
}

// WithEventPlugins installs event plugins for PostgreSQL logical decoding extensions.
func WithEventPlugins(plugins ...types.EventPlugin) Option {
	return decoder.WithEventPlugins(plugins...)
}
