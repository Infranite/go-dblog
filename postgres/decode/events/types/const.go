package types

const (
	// Driver is the PostgreSQL-family backend driver name.
	Driver = "pg"

	// KindBegin is a PostgreSQL transaction begin record.
	KindBegin = "begin"
	// KindCommit is a PostgreSQL transaction commit record.
	KindCommit = "commit"
	// OperationInsert is a PostgreSQL insert record.
	OperationInsert = "insert"
	// OperationUpdate is a PostgreSQL update record.
	OperationUpdate = "update"
	// OperationDelete is a PostgreSQL delete record.
	OperationDelete = "delete"
)

const (
	sqlNullLiteral  = "NULL"
	sqlTrueLiteral  = "TRUE"
	sqlFalseLiteral = "FALSE"
	sqlIsNull       = " IS NULL"
	sqlEquals       = " = "
	sqlAnd          = " AND "
	sqlDot          = "."
	sqlQuote        = "'"
	sqlDQuote       = `"`
	sqlDQuote2      = `""`
)
