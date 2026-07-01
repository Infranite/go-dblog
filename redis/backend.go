package redis

import (
	"github.com/Infranite/go-dblog"
	backendpkg "github.com/Infranite/go-dblog/redis/backend"
)

// Backend opens Redis AOF RESP decoders.
type Backend = backendpkg.Backend

// Register adds Backend to a registry, or to dblog.DefaultRegistry when nil.
func Register(registry *dblog.Registry) error {
	return backendpkg.Register(registry)
}
