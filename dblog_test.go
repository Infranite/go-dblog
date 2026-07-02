package dblog

import (
	"context"
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

type reverseEvent struct {
	testEvent
	reverse any
	ok      bool
}

func (e reverseEvent) Reverse() (any, bool) { return e.reverse, e.ok }

type positionedReverseEvent struct {
	reverseEvent
	position string
}

func (e positionedReverseEvent) PositionString() string { return e.position }

type kindEvent struct {
	testEvent
	kind string
}

func (e kindEvent) Kind() string { return e.kind }

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

func TestCheckpointOfAndOpenOptions(t *testing.T) {
	checkpoint := CheckpointOf(testEvent{body: "query"})
	if checkpoint.Source != (Source{Driver: "test", Name: "fixture"}) {
		t.Fatalf("source = %#v", checkpoint.Source)
	}
	if checkpoint.Position != (Position{Driver: "test", Value: "1"}) {
		t.Fatalf("position = %#v", checkpoint.Position)
	}

	options := newOpenOptions(WithCheckpoint(checkpoint))
	if got := options.Source(); got != checkpoint.Source {
		t.Fatalf("source option = %#v", got)
	}
	if got := StartPositionOf(options); got != checkpoint.Position {
		t.Fatalf("start position = %#v", got)
	}
}

func TestOpenOptionsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	options := newOpenOptions(WithContext(ctx))
	if got := ContextOf(options); got != ctx {
		t.Fatalf("context = %v, want %v", got, ctx)
	}

	if got := ContextOf(nil); got == nil {
		t.Fatal("context is nil")
	}

	options = newOpenOptions()
	if got := ContextOf(options); got == nil {
		t.Fatal("empty options returned nil context")
	}
}

func TestCheckpointOfNilEvent(t *testing.T) {
	if got := CheckpointOf(nil); got != (Checkpoint{}) {
		t.Fatalf("CheckpointOf(nil) = %#v", got)
	}
}

type registryBackend struct {
	driver string
	events []Event
}

func (b registryBackend) Driver() string { return b.driver }

func (b registryBackend) Open(OpenOptions) (Decoder[Event], error) {
	return NewSeqDecoder(func(yield func(Event, error) bool) {
		for _, event := range b.events {
			if !yield(event, nil) {
				return
			}
		}
	}, nil), nil
}

func TestRegistryOpensBackend(t *testing.T) {
	var registry Registry
	err := registry.Register(registryBackend{
		driver: "mongo",
		events: []Event{
			testEvent{body: "insert"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	decoder, err := registry.Open("mongo")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := decoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	var got []any
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, event.Body())
	}
	if len(got) != 1 || got[0] != "insert" {
		t.Fatalf("events = %#v", got)
	}
}

func TestRegistryRejectsDuplicateBackend(t *testing.T) {
	var registry Registry
	if err := registry.Register(registryBackend{driver: "redis"}); err != nil {
		t.Fatal(err)
	}
	err := registry.Register(registryBackend{driver: "redis"})
	if !errors.Is(err, ErrBackendExists) {
		t.Fatalf("err = %v, want %v", err, ErrBackendExists)
	}
}

func TestRegistryRejectsInvalidBackend(t *testing.T) {
	var registry Registry
	err := registry.Register(registryBackend{})
	if !errors.Is(err, ErrInvalidBackend) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidBackend)
	}
}

func TestRegistryReportsUnknownBackend(t *testing.T) {
	var registry Registry
	_, err := registry.Open("pg")
	if !errors.Is(err, ErrBackendNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrBackendNotFound)
	}
}

func TestOpenOptionsUseFunctionalOptions(t *testing.T) {
	options := newOpenOptions(
		WithSource(Source{Driver: "redis", Name: "appendonly.aof"}),
		WithPath("appendonly.aof"),
		WithDSN("redis://localhost"),
	)
	if options.Source() != (Source{Driver: "redis", Name: "appendonly.aof"}) {
		t.Fatalf("source = %#v", options.Source())
	}
	if options.Path() != "appendonly.aof" || options.DSN() != "redis://localhost" {
		t.Fatalf("path/dsn = %q/%q", options.Path(), options.DSN())
	}
}

func TestFilterAppliesPredicates(t *testing.T) {
	seq := func(yield func(Event, error) bool) {
		yield(testEvent{body: "query"}, nil)
		yield(kindEvent{testEvent: testEvent{body: "insert"}, kind: "other"}, nil)
	}

	var got []string
	for event, err := range Filter(seq, ByDriver("test"), ByKind("fixture")) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, event.Body().(string))
	}
	if len(got) != 1 || got[0] != "query" {
		t.Fatalf("Filter returned %#v", got)
	}
}

func TestFlashbacksYieldsReverseOperations(t *testing.T) {
	seq := func(yield func(Event, error) bool) {
		yield(testEvent{body: "ignored"}, nil)
		yield(reverseEvent{testEvent: testEvent{body: "delete"}, reverse: "insert back", ok: true}, nil)
	}

	var got []any
	for op, err := range Flashbacks(seq) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, op)
	}
	if len(got) != 1 || got[0] != "insert back" {
		t.Fatalf("Flashbacks returned %#v", got)
	}
}

func TestRecoveryPlanYieldsOperationsWithCheckpoints(t *testing.T) {
	seq := func(yield func(Event, error) bool) {
		yield(testEvent{body: "ignored"}, nil)
		yield(positionedReverseEvent{
			reverseEvent: reverseEvent{
				testEvent: testEvent{body: "delete"},
				reverse:   "insert back",
				ok:        true,
			},
			position: "42",
		}, nil)
		yield(reverseEvent{testEvent: testEvent{body: "unsafe"}, ok: false}, nil)
	}

	var got []RecoveryStep
	for step, err := range RecoveryPlan(seq) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, step)
	}
	if len(got) != 1 {
		t.Fatalf("RecoveryPlan returned %#v", got)
	}
	if got[0].Operation != "insert back" {
		t.Fatalf("operation = %#v", got[0].Operation)
	}
	wantCheckpoint := Checkpoint{
		Source:   Source{Driver: "test", Name: "fixture"},
		Position: Position{Driver: "test", Value: "42"},
	}
	if got[0].Checkpoint != wantCheckpoint {
		t.Fatalf("checkpoint = %#v, want %#v", got[0].Checkpoint, wantCheckpoint)
	}
}

func TestRecoveryPlanStopsOnError(t *testing.T) {
	wantErr := errors.New("decode")
	seq := func(yield func(Event, error) bool) {
		if !yield(reverseEvent{testEvent: testEvent{body: "delete"}, reverse: "insert back", ok: true}, nil) {
			return
		}
		if !yield(nil, wantErr) {
			return
		}
		yield(reverseEvent{testEvent: testEvent{body: "ignored"}, reverse: "ignored", ok: true}, nil)
	}

	var got []RecoveryStep
	var gotErr error
	for step, err := range RecoveryPlan(seq) {
		if err != nil {
			gotErr = err
			break
		}
		got = append(got, step)
	}
	if !errors.Is(gotErr, wantErr) {
		t.Fatalf("err = %v, want %v", gotErr, wantErr)
	}
	if len(got) != 1 || got[0].Operation != "insert back" {
		t.Fatalf("RecoveryPlan returned %#v", got)
	}
}
