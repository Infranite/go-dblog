package types

const (
	// Driver is the MongoDB-family backend driver name.
	Driver = "mongo"

	// OperationInsert is a MongoDB insert event.
	OperationInsert = "insert"
	// OperationUpdate is a MongoDB update event.
	OperationUpdate = "update"
	// OperationReplace is a MongoDB replacement event.
	OperationReplace = "replace"
	// OperationDelete is a MongoDB delete event.
	OperationDelete = "delete"
	// OperationCommand is a MongoDB command event.
	OperationCommand = "command"
	// OperationNoop is a MongoDB no-op event.
	OperationNoop = "noop"

	// CommandInsert is a MongoDB insert command emitted for flashback output.
	CommandInsert = OperationInsert
	// CommandDelete is a MongoDB delete command emitted for flashback output.
	CommandDelete = OperationDelete
	// CommandReplace is a MongoDB replacement command emitted for flashback output.
	CommandReplace = "replace"
)
