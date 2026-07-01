package types

import (
	"fmt"
	"strings"
	"unicode"
)

func insertSQL(change Change) string {
	names := make([]string, 0, len(change.Columns))
	values := make([]string, 0, len(change.Columns))
	for _, column := range change.Columns {
		names = append(names, quoteIdent(column.Name))
		values = append(values, sqlValue(column.Value))
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);",
		tableName(change), strings.Join(names, ", "), strings.Join(values, ", "))
}

func deleteSQL(change Change) string {
	return fmt.Sprintf("DELETE FROM %s WHERE %s;", tableName(change), whereSQL(change.Columns))
}

func updateSQL(change Change) string {
	set := make([]string, 0, len(change.OldKey))
	for _, column := range change.OldKey {
		set = append(set, quoteIdent(column.Name)+sqlEquals+sqlValue(column.Value))
	}
	return fmt.Sprintf("UPDATE %s SET %s WHERE %s;",
		tableName(change), strings.Join(set, ", "), whereSQL(change.NewTuple))
}

func whereSQL(columns []Column) string {
	where := make([]string, 0, len(columns))
	for _, column := range columns {
		name := quoteIdent(column.Name)
		if column.Value == nil {
			where = append(where, name+sqlIsNull)
			continue
		}
		where = append(where, name+sqlEquals+sqlValue(column.Value))
	}
	return strings.Join(where, sqlAnd)
}

func tableName(change Change) string {
	if change.Schema == "" {
		return quoteIdent(change.Table)
	}
	return quoteIdent(change.Schema) + sqlDot + quoteIdent(change.Table)
}

func quoteIdent(s string) string {
	if s != "" && (unicode.IsLetter(rune(s[0])) || s[0] == '_') {
		ok := true
		for _, r := range s[1:] {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				ok = false
				break
			}
		}
		if ok {
			return s
		}
	}
	return sqlDQuote + strings.ReplaceAll(s, sqlDQuote, sqlDQuote2) + sqlDQuote
}

func sqlValue(v any) string {
	switch value := v.(type) {
	case nil:
		return sqlNullLiteral
	case bool:
		if value {
			return sqlTrueLiteral
		}
		return sqlFalseLiteral
	case string:
		return sqlQuote + strings.ReplaceAll(value, sqlQuote, sqlQuote+sqlQuote) + sqlQuote
	default:
		return fmt.Sprint(value)
	}
}
