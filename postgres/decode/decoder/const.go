package decoder

const (
	linePrefixBegin  = "BEGIN"
	linePrefixCommit = "COMMIT"
	linePrefixTable  = "table "
	fieldSeparator   = ": "
	oldKeyPrefix     = "old-key: "
	newTuplePrefix   = "new-tuple: "
	nullLiteral      = "null"
	trueLiteral      = "true"
	falseLiteral     = "false"
)
