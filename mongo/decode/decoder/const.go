package decoder

const (
	fieldOperationType            = "operationType"
	fieldNamespace                = "ns"
	fieldNamespaceDatabase        = "db"
	fieldNamespaceCollection      = "coll"
	fieldFullDocument             = "fullDocument"
	fieldFullDocumentBeforeChange = "fullDocumentBeforeChange"
	fieldDocumentKey              = "documentKey"
	fieldUpdateDescription        = "updateDescription"
	fieldOplogOperation           = "op"
	fieldOplogObject              = "o"
	fieldOplogObject2             = "o2"
	fieldID                       = "_id"
)

const (
	oplogInsert  = "i"
	oplogUpdate  = "u"
	oplogDelete  = "d"
	oplogCommand = "c"
	oplogNoop    = "n"
)
