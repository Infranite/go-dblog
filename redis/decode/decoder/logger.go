package decoder

import "github.com/Infranite/go-dblog"

// Log is the package-global logger. Use Log.SetLogger to replace it and
// Log.SetLevel to change verbosity.
var Log = dblog.NewLoggerSlot("go-dblog/redis/decode/decoder")
