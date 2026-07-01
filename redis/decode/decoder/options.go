package decoder

import "github.com/Infranite/go-dblog/redis/decode/events/types"

// Option configures a Redis decoder.
type Option func(*options)

type options struct {
	commandPlugins []types.CommandPlugin
	startPosition  int
}

// WithCommandPlugins installs command plugins for Redis dialect extensions.
func WithCommandPlugins(plugins ...types.CommandPlugin) Option {
	return func(o *options) {
		o.commandPlugins = append(o.commandPlugins, plugins...)
	}
}

// WithStartPosition skips commands up to and including position.
func WithStartPosition(position int) Option {
	return func(o *options) {
		o.startPosition = position
	}
}
