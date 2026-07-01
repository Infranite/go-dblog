package decoder

import "github.com/Infranite/go-dblog/redis/decode/events/types"

// Option configures a Redis decoder.
type Option func(*options)

type options struct {
	commandPlugins []types.CommandPlugin
}

// WithCommandPlugins installs command plugins for Redis dialect extensions.
func WithCommandPlugins(plugins ...types.CommandPlugin) Option {
	return func(o *options) {
		o.commandPlugins = append(o.commandPlugins, plugins...)
	}
}
