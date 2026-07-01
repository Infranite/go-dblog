package types

// Change is one decoded MongoDB oplog or change stream event.
type Change struct {
	Operation      string
	Database       string
	Collection     string
	Document       map[string]any
	BeforeDocument map[string]any
	DocumentKey    map[string]any
	Update         map[string]any
	Raw            map[string]any
}

// Command is a MongoDB operation emitted for flashback output.
type Command struct {
	Operation  string
	Database   string
	Collection string
	Filter     map[string]any
	Document   map[string]any
	Update     map[string]any
}

// EventPlugin extends MongoDB change decoding for product-specific events.
type EventPlugin interface {
	Name() string
	Match(map[string]any) bool
	Apply(*Change) error
}
