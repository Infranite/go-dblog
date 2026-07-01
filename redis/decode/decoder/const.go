package decoder

const (
	respArrayPrefix = '*'
	respBulkPrefix  = '$'
	respCR          = '\r'
	respLF          = '\n'
	respLineEnd     = "\r\n"
)

const (
	maxRESPArrayElements   = 8192
	maxRESPBulkStringBytes = 8 << 20
)
