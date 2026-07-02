package decoder

import (
	"testing"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mysql/common"
	"github.com/Infranite/go-dblog/mysql/decode/events"
	"github.com/Infranite/go-dblog/mysql/decode/events/types"
)

func TestDblogDecoderEvents(t *testing.T) {
	fileDecoder, err := NewBinFileDecoder(requireTestBinlog(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := fileDecoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	decoder := WrapDblogDecoder(Source{Name: "mysql-bin.000004"}, fileDecoder)
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		if event.SourceDriver() != "mysql" || event.SourceName() != "mysql-bin.000004" {
			t.Fatalf("source = %s/%s", event.SourceDriver(), event.SourceName())
		}
		if event.Kind() != "FORMAT_DESCRIPTION_EVENT" {
			t.Fatalf("kind = %s", event.Kind())
		}
		if event.PositionDriver() != "mysql" || event.PositionString() == "" {
			t.Fatalf("position = %s/%s", event.PositionDriver(), event.PositionString())
		}
		if _, ok := event.Body().(*types.FmtDescEvent); !ok {
			t.Fatalf("body = %T", event.Body())
		}
		if len(event.Raw()) == 0 {
			t.Fatal("raw event is empty")
		}
		return
	}
	t.Fatal("no events")
}

func TestDblogBodiesFiltersTypedBody(t *testing.T) {
	decoder, err := NewDblogDecoder(requireTestBinlog(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := decoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		body, ok := event.Body().(*types.QueryEvent)
		if !ok {
			continue
		}
		if body.Query == "" {
			t.Fatal("query event has empty query")
		}
		return
	}
	t.Fatal("query event not found")
}

func TestDblogFilterAndFlashbacks(t *testing.T) {
	decoder, err := NewDblogDecoder(requireTestBinlog(t))
	if err != nil {
		t.Fatal(err)
	}
	queryDecoder := decoder
	t.Cleanup(func() {
		if err := queryDecoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	var queries int
	for event, err := range dblog.Filter(dblog.Events(decoder), dblog.ByKind("QUERY_EVENT")) {
		if err != nil {
			t.Fatal(err)
		}
		if event.SourceDriver() != "mysql" {
			t.Fatalf("driver = %s", event.SourceDriver())
		}
		queries++
	}
	if queries == 0 {
		t.Fatal("query events are empty")
	}

	decoder, err = NewDblogDecoder(requireTestBinlog(t))
	if err != nil {
		t.Fatal(err)
	}
	flashbackDecoder := decoder
	t.Cleanup(func() {
		if err := flashbackDecoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	var flashbacks int
	for op, err := range dblog.Flashbacks(dblog.Events(decoder)) {
		if err != nil {
			t.Fatal(err)
		}
		event, ok := op.(*events.Event)
		if !ok {
			t.Fatalf("flashback = %T, want *events.Event", op)
		}
		rows, ok := event.Body.(*types.BinRowsEvent)
		if !ok || rows.TableID == 0 {
			t.Fatalf("flashback body = %#v", event.Body)
		}
		flashbacks++
	}
	if flashbacks == 0 {
		t.Fatal("fixture produced no mysql flashback operations")
	}
}

func TestDblogRecoveryPlanIncludesCheckpoint(t *testing.T) {
	decoder, err := NewDblogDecoder(requireTestBinlog(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := decoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	for step, err := range dblog.RecoveryPlan(dblog.Events(decoder)) {
		if err != nil {
			t.Fatal(err)
		}
		if step.Checkpoint.Source.Driver != "mysql" || step.Checkpoint.Position.Value == "" {
			t.Fatalf("checkpoint = %#v", step.Checkpoint)
		}
		event, ok := step.Operation.(*events.Event)
		if !ok {
			t.Fatalf("operation = %T, want *events.Event", step.Operation)
		}
		if _, ok := event.Body.(*types.BinRowsEvent); !ok {
			t.Fatalf("operation body = %T, want *types.BinRowsEvent", event.Body)
		}
		return
	}
	t.Fatal("fixture produced no mysql recovery steps")
}

func TestDblogEventReverseRowsEvents(t *testing.T) {
	row := []types.ColumnValue{{Type: common.MySQLTypeLong, Value: int64(1)}}
	before := []types.ColumnValue{{Type: common.MySQLTypeLong, Value: int64(1)}}
	after := []types.ColumnValue{{Type: common.MySQLTypeLong, Value: int64(2)}}

	tests := []struct {
		name      string
		eventType uint8
		body      *types.BinRowsEvent
		wantType  uint8
	}{
		{
			name:      "write rows reverse to delete rows",
			eventType: common.WriteRowsEventV2,
			body: &types.BinRowsEvent{
				TableID:     7,
				Schema:      "dblog_ci",
				Table:       "events",
				ColumnCount: 1,
				Rows:        [][]types.ColumnValue{row},
			},
			wantType: common.DeleteRowsEventV2,
		},
		{
			name:      "delete rows reverse to write rows",
			eventType: common.DeleteRowsEventV2,
			body: &types.BinRowsEvent{
				TableID:     7,
				Schema:      "dblog_ci",
				Table:       "events",
				ColumnCount: 1,
				Rows:        [][]types.ColumnValue{row},
			},
			wantType: common.WriteRowsEventV2,
		},
		{
			name:      "update rows swaps before and after images",
			eventType: common.UpdateRowsEventV2,
			body: &types.BinRowsEvent{
				TableID:     7,
				Schema:      "dblog_ci",
				Table:       "events",
				ColumnCount: 1,
				BeforeRows:  [][]types.ColumnValue{before},
				AfterRows:   [][]types.ColumnValue{after},
			},
			wantType: common.UpdateRowsEventV2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := DblogEvent{event: &events.Event{
				Header: &events.EventHeader{EventType: tt.eventType},
				Body:   tt.body,
			}}

			got, ok := event.Reverse()
			if !ok {
				t.Fatal("expected mysql row flashback")
			}
			reverseEvent, ok := got.(*events.Event)
			if !ok {
				t.Fatalf("flashback = %T, want *events.Event", got)
			}
			if reverseEvent.Header.EventType != tt.wantType {
				t.Fatalf("reverse event type = %s, want %s",
					common.EventTypeName(reverseEvent.Header.EventType),
					common.EventTypeName(tt.wantType))
			}
			rows, ok := reverseEvent.Body.(*types.BinRowsEvent)
			if !ok {
				t.Fatalf("reverse body = %T, want *types.BinRowsEvent", reverseEvent.Body)
			}
			if rows.Schema != tt.body.Schema || rows.Table != tt.body.Table || rows.TableID != tt.body.TableID {
				t.Fatalf("reverse table = %s.%s/%d", rows.Schema, rows.Table, rows.TableID)
			}
			if tt.eventType == common.UpdateRowsEventV2 {
				if rows.BeforeRows[0][0].Value != int64(2) || rows.AfterRows[0][0].Value != int64(1) {
					t.Fatalf("reverse update rows = before %#v after %#v", rows.BeforeRows, rows.AfterRows)
				}
			}
		})
	}
}

func TestDblogEventReverseSkipsUnsafeRowsEvents(t *testing.T) {
	tests := []struct {
		name      string
		eventType uint8
		body      *types.BinRowsEvent
	}{
		{
			name:      "missing table map",
			eventType: common.WriteRowsEventV2,
			body: &types.BinRowsEvent{
				TableID:     7,
				ColumnCount: 1,
				Rows:        [][]types.ColumnValue{{{Type: common.MySQLTypeLong, Value: int64(1)}}},
				DecodeError: "missing table map",
			},
		},
		{
			name:      "empty row image",
			eventType: common.WriteRowsEventV2,
			body: &types.BinRowsEvent{
				TableID:     7,
				Schema:      "dblog_ci",
				Table:       "events",
				ColumnCount: 1,
			},
		},
		{
			name:      "partial row image",
			eventType: common.WriteRowsEventV2,
			body: &types.BinRowsEvent{
				TableID:     7,
				Schema:      "dblog_ci",
				Table:       "events",
				ColumnCount: 2,
				Rows: [][]types.ColumnValue{{
					{Type: common.MySQLTypeLong, Value: int64(1)},
					{Type: common.MySQLTypeVarchar, Skipped: true},
				}},
			},
		},
		{
			name:      "partial update event",
			eventType: common.PartialUpdateRowsEvent,
			body: &types.BinRowsEvent{
				TableID:     7,
				Schema:      "dblog_ci",
				Table:       "events",
				ColumnCount: 1,
				BeforeRows:  [][]types.ColumnValue{{{Type: common.MySQLTypeLong, Value: int64(1)}}},
				AfterRows:   [][]types.ColumnValue{{{Type: common.MySQLTypeLong, Value: int64(2)}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := DblogEvent{event: &events.Event{
				Header: &events.EventHeader{EventType: tt.eventType},
				Body:   tt.body,
			}}
			if got, ok := event.Reverse(); ok {
				t.Fatalf("reverse = %#v, want none", got)
			}
		})
	}
}
