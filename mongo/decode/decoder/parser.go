package decoder

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo/decode/events/types"
)

// ParseLine parses one MongoDB oplog or change stream JSON line.
func ParseLine(source dblog.Source, position int, line string) (types.Event, error) {
	return parseLine(source, position, line, nil)
}

func parseLine(
	source dblog.Source,
	position int,
	line string,
	plugins []types.EventPlugin,
) (types.Event, error) {
	if source.Driver == "" {
		source.Driver = types.Driver
	}
	var raw map[string]any
	decoder := json.NewDecoder(strings.NewReader(line))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return types.Event{}, fmt.Errorf("%w: %v", types.ErrInvalidJSON, err)
	}
	change, err := parseChange(raw)
	if err != nil {
		return types.Event{}, err
	}
	if err := applyEventPlugins(raw, &change, plugins); err != nil {
		return types.Event{}, err
	}
	return types.NewEvent(source, position, []byte(line), change), nil
}

func parseChange(raw map[string]any) (types.Change, error) {
	change := types.Change{Raw: raw}
	if op, ok := raw[fieldOperationType].(string); ok {
		change.Operation = strings.ToLower(op)
		if ns := asMap(raw[fieldNamespace]); ns != nil {
			change.Database = stringValue(ns[fieldNamespaceDatabase])
			change.Collection = stringValue(ns[fieldNamespaceCollection])
		}
		change.Document = asMap(raw[fieldFullDocument])
		if before := asMap(raw[fieldFullDocumentBeforeChange]); before != nil && change.Document == nil {
			change.Document = before
		}
		change.DocumentKey = asMap(raw[fieldDocumentKey])
		change.Update = asMap(raw[fieldUpdateDescription])
		return change, nil
	}

	op, _ := raw[fieldOplogOperation].(string)
	change.Operation = oplogOperation(op)
	db, coll := splitNamespace(stringValue(raw[fieldNamespace]))
	change.Database = db
	change.Collection = coll
	change.Document = asMap(raw[fieldOplogObject])
	change.DocumentKey = asMap(raw[fieldOplogObject2])
	if change.DocumentKey == nil && change.Document != nil {
		if id, ok := change.Document[fieldID]; ok {
			change.DocumentKey = map[string]any{fieldID: id}
		}
	}
	if change.Operation == "" {
		return types.Change{}, fmt.Errorf("%w: %q", types.ErrUnsupportedOperation, op)
	}
	return change, nil
}

func applyEventPlugins(
	raw map[string]any,
	change *types.Change,
	plugins []types.EventPlugin,
) error {
	for _, plugin := range plugins {
		if plugin == nil || !plugin.Match(raw) {
			continue
		}
		if err := plugin.Apply(change); err != nil {
			return fmt.Errorf("apply mongo event plugin %s: %w", plugin.Name(), err)
		}
	}
	return nil
}

func oplogOperation(op string) string {
	switch op {
	case oplogInsert:
		return types.OperationInsert
	case oplogUpdate:
		return types.OperationUpdate
	case oplogDelete:
		return types.OperationDelete
	case oplogCommand:
		return types.OperationCommand
	case oplogNoop:
		return types.OperationNoop
	default:
		return ""
	}
}

func splitNamespace(ns string) (string, string) {
	if i := strings.IndexByte(ns, '.'); i >= 0 {
		return ns[:i], ns[i+1:]
	}
	return ns, ""
}

func asMap(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return nil
	}
	return m
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}
