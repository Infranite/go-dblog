package decoder

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo/decode/events/types"
	"go.mongodb.org/mongo-driver/v2/bson"
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
	var raw map[string]any
	decoder := json.NewDecoder(strings.NewReader(line))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return types.Event{}, fmt.Errorf("%w: %v", types.ErrInvalidJSON, err)
	}
	return parseRaw(source, position, []byte(line), raw, plugins)
}

func parseRaw(
	source dblog.Source,
	position int,
	rawBytes []byte,
	raw map[string]any,
	plugins []types.EventPlugin,
) (types.Event, error) {
	if source.Driver == "" {
		source.Driver = types.Driver
	}
	change, err := parseChange(raw)
	if err != nil {
		return types.Event{}, err
	}
	if err := applyEventPlugins(raw, &change, plugins); err != nil {
		return types.Event{}, err
	}
	if rawBytes == nil {
		rawBytes, _ = json.Marshal(raw)
	}
	return types.NewEvent(source, position, rawBytes, change), nil
}

func parseChange(raw map[string]any) (types.Change, error) {
	change := types.Change{Raw: raw}
	if op, ok := raw[fieldOperationType].(string); ok {
		change.Operation = strings.ToLower(strings.TrimSpace(op))
		if change.Operation == "" {
			return types.Change{}, fmt.Errorf("%w: %q", types.ErrUnsupportedOperation, op)
		}
		if ns := asMap(raw[fieldNamespace]); ns != nil {
			change.Database = stringValue(ns[fieldNamespaceDatabase])
			change.Collection = stringValue(ns[fieldNamespaceCollection])
		}
		change.Document = asMap(raw[fieldFullDocument])
		change.BeforeDocument = asMap(raw[fieldFullDocumentBeforeChange])
		if change.BeforeDocument != nil && change.Document == nil {
			change.Document = change.BeforeDocument
		}
		change.DocumentKey = asMap(raw[fieldDocumentKey])
		if update, err := optionalMap(raw, fieldUpdateDescription); err != nil {
			return types.Change{}, err
		} else {
			change.Update = update
		}
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
	switch x := v.(type) {
	case map[string]any:
		if len(x) == 0 {
			return nil
		}
		return x
	case bson.M:
		return normalizeMap(map[string]any(x))
	case bson.D:
		return normalizeD(x)
	default:
		return nil
	}
}

func optionalMap(raw map[string]any, field string) (map[string]any, error) {
	value, ok := raw[field]
	if !ok || value == nil {
		return nil, nil
	}
	switch x := value.(type) {
	case map[string]any:
		return normalizeMapKeepEmpty(x), nil
	case bson.M:
		return normalizeMapKeepEmpty(map[string]any(x)), nil
	case bson.D:
		return normalizeDKeepEmpty(x), nil
	default:
		return nil, fmt.Errorf("%w: %s must be an object", types.ErrInvalidJSON, field)
	}
}

func normalizeMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = normalizeValue(v)
	}
	return out
}

func normalizeMapKeepEmpty(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = normalizeValue(v)
	}
	return out
}

func normalizeD(in bson.D) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for _, elem := range in {
		out[elem.Key] = normalizeValue(elem.Value)
	}
	return out
}

func normalizeDKeepEmpty(in bson.D) map[string]any {
	out := make(map[string]any, len(in))
	for _, elem := range in {
		out[elem.Key] = normalizeValue(elem.Value)
	}
	return out
}

func normalizeValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return x
	case bson.M:
		return normalizeMap(map[string]any(x))
	case bson.D:
		return normalizeD(x)
	case bson.A:
		out := make([]any, len(x))
		for i, elem := range x {
			out[i] = normalizeValue(elem)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, elem := range x {
			out[i] = normalizeValue(elem)
		}
		return out
	default:
		return v
	}
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}
