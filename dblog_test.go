package dblog

import (
	"errors"
	"iter"
	"testing"
)

type testEvent struct {
	body any
}

func (e testEvent) SourceDriver() string   { return "test" }
func (e testEvent) SourceName() string     { return "fixture" }
func (e testEvent) PositionDriver() string { return "test" }
func (e testEvent) PositionString() string { return "1" }
func (e testEvent) Kind() string           { return "fixture" }
func (e testEvent) Raw() []byte            { return []byte("raw") }
func (e testEvent) Body() any              { return e.body }

type testDecoder struct{}

func (d testDecoder) Events() iter.Seq2[testEvent, error] {
	return func(yield func(testEvent, error) bool) {
		yield(testEvent{body: "query"}, nil)
	}
}

func (d testDecoder) Close() error { return nil }

type testDecoderSeq struct {
	seq iter.Seq2[testEvent, error]
}

func (d testDecoderSeq) Events() iter.Seq2[testEvent, error] { return d.seq }
func (d testDecoderSeq) Close() error                        { return nil }

func TestBodiesFiltersEventBodyType(t *testing.T) {
	seq := func(yield func(Event, error) bool) {
		yield(testEvent{body: "query"}, nil)
		yield(testEvent{body: 1}, nil)
		yield(nil, nil)
	}

	var got []string
	for body, err := range Bodies[string](seq) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, body)
	}
	if len(got) != 1 || got[0] != "query" {
		t.Fatalf("Bodies returned %#v", got)
	}
}

func TestBodiesStopsOnError(t *testing.T) {
	wantErr := errors.New("decode")
	seq := func(yield func(Event, error) bool) {
		if !yield(testEvent{body: "query"}, nil) {
			return
		}
		if !yield(nil, wantErr) {
			return
		}
		yield(testEvent{body: "ignored"}, nil)
	}

	var got []string
	var gotErr error
	for body, err := range Bodies[string](seq) {
		if err != nil {
			gotErr = err
			break
		}
		got = append(got, body)
	}
	if !errors.Is(gotErr, wantErr) {
		t.Fatalf("err = %v, want %v", gotErr, wantErr)
	}
	if len(got) != 1 || got[0] != "query" {
		t.Fatalf("Bodies returned %#v", got)
	}
}

func TestBodiesStopsWhenYieldReturnsFalse(t *testing.T) {
	seq := func(yield func(Event, error) bool) {
		if !yield(testEvent{body: "first"}, nil) {
			return
		}
		yield(testEvent{body: "second"}, nil)
	}

	var got []string
	for body := range Bodies[string](seq) {
		got = append(got, body)
		break
	}
	if len(got) != 1 || got[0] != "first" {
		t.Fatalf("Bodies returned %#v", got)
	}
}

func TestEventsAdaptsTypedDecoder(t *testing.T) {
	var count int
	for event, err := range Events(testDecoder{}) {
		if err != nil {
			t.Fatal(err)
		}
		if event.Kind() != "fixture" {
			t.Fatalf("kind = %s", event.Kind())
		}
		count++
	}
	if count != 1 {
		t.Fatalf("count = %d", count)
	}
}

func TestEventsHandlesNilDecoder(t *testing.T) {
	var got []Event
	for event := range Events[testEvent](nil) {
		got = append(got, event)
	}
	if len(got) != 0 {
		t.Fatalf("Events returned %#v", got)
	}
}

func TestEventsStopsOnError(t *testing.T) {
	wantErr := errors.New("decode")
	decoder := testDecoderSeq{
		seq: func(yield func(testEvent, error) bool) {
			if !yield(testEvent{body: "query"}, nil) {
				return
			}
			if !yield(testEvent{}, wantErr) {
				return
			}
			yield(testEvent{body: "ignored"}, nil)
		},
	}

	var count int
	var gotErr error
	for _, err := range Events(decoder) {
		if err != nil {
			gotErr = err
			break
		}
		count++
	}
	if !errors.Is(gotErr, wantErr) {
		t.Fatalf("err = %v, want %v", gotErr, wantErr)
	}
	if count != 1 {
		t.Fatalf("count = %d", count)
	}
}

func TestEventsStopsWhenYieldReturnsFalse(t *testing.T) {
	decoder := testDecoderSeq{
		seq: func(yield func(testEvent, error) bool) {
			if !yield(testEvent{body: "first"}, nil) {
				return
			}
			yield(testEvent{body: "second"}, nil)
		},
	}

	var got []Event
	for event := range Events(decoder) {
		got = append(got, event)
		break
	}
	if len(got) != 1 || got[0].Body() != "first" {
		t.Fatalf("Events returned %#v", got)
	}
}

func TestSourceAndPosition(t *testing.T) {
	event := testEvent{body: "query"}
	if got := SourceOf(event); got.Driver != "test" || got.Name != "fixture" {
		t.Fatalf("SourceOf = %#v", got)
	}
	if got := PositionOf(event); got.Driver != "test" || got.Value != "1" {
		t.Fatalf("PositionOf = %#v", got)
	}
}

func TestSourceAndPositionHandleNilEvent(t *testing.T) {
	if got := SourceOf(nil); got != (Source{}) {
		t.Fatalf("SourceOf(nil) = %#v", got)
	}
	if got := PositionOf(nil); got != (Position{}) {
		t.Fatalf("PositionOf(nil) = %#v", got)
	}
}
