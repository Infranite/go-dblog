package types

// Column is one decoded column value from a logical decoding change.
type Column struct {
	Name  string
	Type  string
	Value any
	Raw   string
}

// Change is one decoded row-level logical decoding change.
type Change struct {
	Schema    string
	Table     string
	Operation string
	Columns   []Column
	OldKey    []Column
	NewTuple  []Column
}

// Transaction is a BEGIN or COMMIT record.
type Transaction struct {
	Action string
	ID     string
}
