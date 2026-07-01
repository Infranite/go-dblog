package mongo

import (
	"github.com/Infranite/go-dblog"
	backendpkg "github.com/Infranite/go-dblog/mongo/backend"
)

// Backend opens MongoDB JSON change decoders.
type Backend = backendpkg.Backend

// Register adds Backend to a registry, or to dblog.DefaultRegistry when nil.
func Register(registry *dblog.Registry) error {
	return backendpkg.Register(registry)
}
