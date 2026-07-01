package decoder

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres/decode/events/types"
)

// ParseLine parses one PostgreSQL logical decoding text line.
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
	for _, plugin := range plugins {
		if plugin == nil || !plugin.Match(line) {
			continue
		}
		event, err := plugin.Decode(source, position, line)
		if err != nil {
			return types.Event{}, fmt.Errorf("apply postgres event plugin %s: %w", plugin.Name(), err)
		}
		return event, nil
	}
	switch {
	case strings.HasPrefix(line, linePrefixBegin):
		body := types.Transaction{
			Action: types.KindBegin,
			ID:     strings.TrimSpace(strings.TrimPrefix(line, linePrefixBegin)),
		}
		return types.NewEvent(source, position, []byte(line), types.KindBegin, body), nil
	case strings.HasPrefix(line, linePrefixCommit):
		body := types.Transaction{
			Action: types.KindCommit,
			ID:     strings.TrimSpace(strings.TrimPrefix(line, linePrefixCommit)),
		}
		return types.NewEvent(source, position, []byte(line), types.KindCommit, body), nil
	case strings.HasPrefix(line, linePrefixTable):
		change, err := parseChange(line)
		if err != nil {
			return types.Event{}, err
		}
		return types.NewEvent(source, position, []byte(line), change.Operation, change), nil
	default:
		return types.Event{}, fmt.Errorf("%w: %q", types.ErrInvalidLine, line)
	}
}

func parseChange(line string) (types.Change, error) {
	rest := strings.TrimPrefix(line, linePrefixTable)
	i := strings.Index(rest, fieldSeparator)
	if i < 0 {
		return types.Change{}, fmt.Errorf("%w: change %q", types.ErrInvalidLine, line)
	}
	schema, table := splitTable(strings.TrimSpace(rest[:i]))
	if table == "" {
		return types.Change{}, fmt.Errorf("%w: table %q", types.ErrInvalidLine, line)
	}
	rest = rest[i+len(fieldSeparator):]
	i = strings.Index(rest, fieldSeparator)
	if i < 0 {
		return types.Change{}, fmt.Errorf("%w: operation %q", types.ErrInvalidLine, line)
	}
	op := strings.ToLower(strings.TrimSpace(rest[:i]))
	if op == "" {
		return types.Change{}, fmt.Errorf("%w: operation %q", types.ErrInvalidLine, line)
	}
	columns, err := parseColumns(rest[i+len(fieldSeparator):])
	if err != nil {
		return types.Change{}, err
	}
	return types.Change{Schema: schema, Table: table, Operation: op, Columns: columns}, nil
}

func splitTable(name string) (string, string) {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[:i], name[i+1:]
	}
	return "", name
}

func parseColumns(s string) ([]types.Column, error) {
	var columns []types.Column
	for pos := 0; pos < len(s); {
		for pos < len(s) && s[pos] == ' ' {
			pos++
		}
		if pos == len(s) {
			break
		}
		column, next, err := parseColumn(s, pos)
		if err != nil {
			return nil, err
		}
		columns = append(columns, column)
		pos = next
	}
	return columns, nil
}

func parseColumn(s string, pos int) (types.Column, int, error) {
	nameStart := pos
	for pos < len(s) && s[pos] != '[' {
		pos++
	}
	if pos == len(s) {
		return types.Column{}, pos, fmt.Errorf("%w: column %q", types.ErrInvalidLine, s[nameStart:])
	}
	name := s[nameStart:pos]
	pos++
	typeStart := pos
	for pos < len(s) && s[pos] != ']' {
		pos++
	}
	if pos == len(s) || pos+1 >= len(s) || s[pos+1] != ':' {
		return types.Column{}, pos, fmt.Errorf("%w: column type %q", types.ErrInvalidLine, name)
	}
	typ := s[typeStart:pos]
	pos += 2

	raw, value, next, err := parseColumnValue(s, pos)
	if err != nil {
		return types.Column{}, next, err
	}
	return types.Column{Name: name, Type: typ, Value: value, Raw: raw}, next, nil
}

func parseColumnValue(s string, pos int) (string, any, int, error) {
	if pos < len(s) && s[pos] == '\'' {
		return parseQuotedValue(s, pos)
	}

	start := pos
	for pos < len(s) && s[pos] != ' ' {
		pos++
	}
	raw := s[start:pos]
	return raw, parseScalar(raw), pos, nil
}

func parseQuotedValue(s string, pos int) (string, any, int, error) {
	var b strings.Builder
	start := pos
	pos++
	for pos < len(s) {
		if s[pos] == '\'' {
			if pos+1 < len(s) && s[pos+1] == '\'' {
				b.WriteByte('\'')
				pos += 2
				continue
			}
			return s[start : pos+1], b.String(), pos + 1, nil
		}
		b.WriteByte(s[pos])
		pos++
	}
	return "", nil, pos, errors.Join(types.ErrInvalidLine, io.ErrUnexpectedEOF)
}

func parseScalar(raw string) any {
	switch raw {
	case nullLiteral:
		return nil
	case trueLiteral:
		return true
	case falseLiteral:
		return false
	}
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		return f
	}
	return raw
}
