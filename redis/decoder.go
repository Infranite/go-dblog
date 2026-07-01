package redis

import (
	"io"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis/decode/decoder"
	"github.com/Infranite/go-dblog/redis/decode/events/types"
)

// Option configures a Redis decoder.
type Option = decoder.Option

// Decoder streams Redis AOF RESP commands.
type Decoder = decoder.Decoder

// NewDecoder creates a decoder over Redis AOF RESP commands.
func NewDecoder(source dblog.Source, reader io.Reader, close func() error, opts ...Option) *Decoder {
	return decoder.NewDecoder(source, reader, close, opts...)
}

// WithCommandPlugins installs command plugins for Redis dialect extensions.
func WithCommandPlugins(plugins ...types.CommandPlugin) Option {
	return decoder.WithCommandPlugins(plugins...)
}
