package mongo

import (
	"context"
	"io"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo/decode/decoder"
	"github.com/Infranite/go-dblog/mongo/decode/events/types"
)

// Option configures a MongoDB decoder.
type Option = decoder.Option

// Decoder streams MongoDB oplog or change stream JSON lines.
type Decoder = decoder.Decoder

// LiveDecoder streams MongoDB collection change stream events.
type LiveDecoder = decoder.LiveDecoder

// NewDecoder creates a decoder over MongoDB JSON change lines.
func NewDecoder(source dblog.Source, reader io.Reader, close func() error, opts ...Option) *Decoder {
	return decoder.NewDecoder(source, reader, close, opts...)
}

// NewLiveDecoder creates a decoder over a live MongoDB collection change stream.
func NewLiveDecoder(ctx context.Context, source dblog.Source, dsn, namespace string, opts ...Option) (*LiveDecoder, error) {
	return decoder.NewLiveDecoder(ctx, source, dsn, namespace, opts...)
}

// WithEventPlugins installs event plugins for MongoDB dialect extensions.
func WithEventPlugins(plugins ...types.EventPlugin) Option {
	return decoder.WithEventPlugins(plugins...)
}
