package decoder

import "github.com/Infranite/go-dblog/postgres/decode/events/types"

// Option configures a PostgreSQL decoder.
type Option func(*options)

type options struct {
	eventPlugins []types.EventPlugin
}

// WithEventPlugins installs event plugins for PostgreSQL logical decoding extensions.
func WithEventPlugins(plugins ...types.EventPlugin) Option {
	return func(o *options) {
		o.eventPlugins = append(o.eventPlugins, plugins...)
	}
}
