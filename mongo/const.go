package mongo

import "github.com/Infranite/go-dblog/mongo/decode/events/types"

const (
	// Driver is the MongoDB-family backend driver name.
	Driver = types.Driver

	// OperationInsert is a MongoDB insert event.
	OperationInsert = types.OperationInsert
	// OperationUpdate is a MongoDB update event.
	OperationUpdate = types.OperationUpdate
	// OperationReplace is a MongoDB replacement event.
	OperationReplace = types.OperationReplace
	// OperationDelete is a MongoDB delete event.
	OperationDelete = types.OperationDelete
	// OperationCommand is a MongoDB command event.
	OperationCommand = types.OperationCommand
	// OperationNoop is a MongoDB no-op event.
	OperationNoop = types.OperationNoop

	// CommandInsert is a MongoDB insert command emitted for flashback output.
	CommandInsert = types.CommandInsert
	// CommandDelete is a MongoDB delete command emitted for flashback output.
	CommandDelete = types.CommandDelete
	// CommandReplace is a MongoDB replacement command emitted for flashback output.
	CommandReplace = types.CommandReplace
)
