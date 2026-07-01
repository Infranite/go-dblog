package mongo

import "github.com/Infranite/go-dblog/mongo/decode/events/types"

const (
	// Driver is the MongoDB-family backend driver name.
	Driver = types.Driver

	// OperationInsert is a MongoDB insert event.
	OperationInsert = types.OperationInsert
	// OperationUpdate is a MongoDB update event.
	OperationUpdate = types.OperationUpdate
	// OperationDelete is a MongoDB delete event.
	OperationDelete = types.OperationDelete
	// OperationCommand is a MongoDB command event.
	OperationCommand = types.OperationCommand
	// OperationNoop is a MongoDB no-op event.
	OperationNoop = types.OperationNoop
)
