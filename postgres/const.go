package postgres

import "github.com/Infranite/go-dblog/postgres/decode/events/types"

const (
	// Driver is the PostgreSQL-family backend driver name.
	Driver = types.Driver

	// KindBegin is a PostgreSQL transaction begin record.
	KindBegin = types.KindBegin
	// KindCommit is a PostgreSQL transaction commit record.
	KindCommit = types.KindCommit
	// OperationInsert is a PostgreSQL insert record.
	OperationInsert = types.OperationInsert
	// OperationUpdate is a PostgreSQL update record.
	OperationUpdate = types.OperationUpdate
	// OperationDelete is a PostgreSQL delete record.
	OperationDelete = types.OperationDelete
)
