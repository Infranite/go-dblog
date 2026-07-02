package redis

import (
	"context"
	"io"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis/decode/decoder"
	"github.com/Infranite/go-dblog/redis/decode/events/types"
)

// Option configures a Redis decoder.
type Option = decoder.Option

// Decoder streams Redis AOF RESP commands.
type Decoder = decoder.Decoder

// LiveDecoder streams Redis replication commands.
type LiveDecoder = decoder.LiveDecoder

// NewDecoder creates a decoder over Redis AOF RESP commands.
func NewDecoder(source dblog.Source, reader io.Reader, close func() error, opts ...Option) *Decoder {
	return decoder.NewDecoder(source, reader, close, opts...)
}

// NewLiveDecoder creates a decoder over a live Redis replication stream.
func NewLiveDecoder(ctx context.Context, source dblog.Source, dsn string, opts ...Option) (*LiveDecoder, error) {
	return decoder.NewLiveDecoder(ctx, source, dsn, opts...)
}

// WithCommandPlugins installs command plugins for Redis dialect extensions.
func WithCommandPlugins(plugins ...types.CommandPlugin) Option {
	return decoder.WithCommandPlugins(plugins...)
}
