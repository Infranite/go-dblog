package mongo

import (
	"io"

	"github.com/Infranite/go-dblog"
)

type nilOptions struct{}

func (nilOptions) Source() dblog.Source { return dblog.Source{} }
func (nilOptions) Path() string         { return "" }
func (nilOptions) DSN() string          { return "" }
func (nilOptions) Reader() io.Reader    { return nil }
