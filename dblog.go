// Package dblog defines the shared user-facing contracts for database log
// backends.
package dblog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"sync"
)

var (
	// ErrInvalidBackend is returned when a backend does not expose a driver.
	ErrInvalidBackend = errors.New("invalid backend")
	// ErrBackendExists is returned when a registry already has a driver.
	ErrBackendExists = errors.New("backend already registered")
	// ErrBackendNotFound is returned when a driver is not registered.
	ErrBackendNotFound = errors.New("backend not found")
)

// Source identifies the database log producer behind an event stream.
type Source struct {
	Driver string
	Name   string
}

// Position identifies a backend-specific location in a database log stream.
type Position struct {
	Driver string
	Value  string
}

// Checkpoint stores the last consumed event location for a source.
type Checkpoint struct {
	Source   Source
	Position Position
}

// Event is the minimal common shape shared by database log backends.
type Event interface {
	SourceDriver() string
	SourceName() string
	PositionDriver() string
	PositionString() string
	Kind() string
	Raw() []byte
	Body() any
}

// Decoder streams events from one database log source.
type Decoder[T Event] interface {
	Events() iter.Seq2[T, error]
	Close() error
}

// OpenOptions exposes backend decoder creation options to backend packages.
type OpenOptions interface {
	Source() Source
	Path() string
	DSN() string
	Reader() io.Reader
}

type openOptions struct {
	source        Source
	path          string
	dsn           string
	reader        io.Reader
	context       context.Context
	startPosition Position
}

func (o openOptions) Source() Source    { return o.source }
func (o openOptions) Path() string      { return o.path }
func (o openOptions) DSN() string       { return o.dsn }
func (o openOptions) Reader() io.Reader { return o.reader }
func (o openOptions) Context() context.Context {
	if o.context == nil {
		return context.Background()
	}
	return o.context
}
func (o openOptions) StartPosition() Position {
	return o.startPosition
}

// OpenOption configures backend decoder creation.
type OpenOption func(*openOptions)

// WithSource sets the source metadata for opened events.
func WithSource(source Source) OpenOption {
	return func(options *openOptions) {
		options.source = source
	}
}

// WithPath opens a backend from a local path when supported.
func WithPath(path string) OpenOption {
	return func(options *openOptions) {
		options.path = path
	}
}

// WithDSN sets a backend-specific data source name.
func WithDSN(dsn string) OpenOption {
	return func(options *openOptions) {
		options.dsn = dsn
	}
}

// WithReader opens a backend from a stream when supported.
func WithReader(reader io.Reader) OpenOption {
	return func(options *openOptions) {
		options.reader = reader
	}
}

// WithContext sets the cancellation context used by context-aware backends.
func WithContext(ctx context.Context) OpenOption {
	return func(options *openOptions) {
		options.context = ctx
	}
}

// WithCheckpoint resumes an opened backend after a previously consumed event.
func WithCheckpoint(checkpoint Checkpoint) OpenOption {
	return func(options *openOptions) {
		if checkpoint.Source != (Source{}) {
			options.source = checkpoint.Source
		}
		options.startPosition = checkpoint.Position
	}
}

// Backend opens decoders for one database log driver.
type Backend interface {
	Driver() string
	Open(OpenOptions) (Decoder[Event], error)
}

// Registry stores backend drivers.
type Registry struct {
	mu       sync.RWMutex
	backends map[string]Backend
}

// Register adds a backend driver to the registry.
func (r *Registry) Register(backend Backend) error {
	if backend == nil || backend.Driver() == "" {
		return fmt.Errorf("%w: empty driver", ErrInvalidBackend)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.backends == nil {
		r.backends = make(map[string]Backend)
	}
	driver := backend.Driver()
	if _, ok := r.backends[driver]; ok {
		return fmt.Errorf("%w: %s", ErrBackendExists, driver)
	}
	r.backends[driver] = backend
	return nil
}

// Open creates a decoder for a registered driver.
func (r *Registry) Open(driver string, opts ...OpenOption) (Decoder[Event], error) {
	r.mu.RLock()
	backend := r.backends[driver]
	r.mu.RUnlock()
	if backend == nil {
		return nil, fmt.Errorf("%w: %s", ErrBackendNotFound, driver)
	}
	return backend.Open(newOpenOptions(opts...))
}

// DefaultRegistry is used by package-level Register and Open.
var DefaultRegistry Registry

// Register adds a backend to DefaultRegistry.
func Register(backend Backend) error {
	return DefaultRegistry.Register(backend)
}

// Open creates a decoder from DefaultRegistry.
func Open(driver string, opts ...OpenOption) (Decoder[Event], error) {
	return DefaultRegistry.Open(driver, opts...)
}

// SeqDecoder adapts an event iterator to Decoder.
type SeqDecoder struct {
	seq   iter.Seq2[Event, error]
	close func() error
}

// NewSeqDecoder creates a decoder from an event iterator.
func NewSeqDecoder(seq iter.Seq2[Event, error], close func() error) *SeqDecoder {
	return &SeqDecoder{seq: seq, close: close}
}

// Events returns the wrapped event iterator.
func (d *SeqDecoder) Events() iter.Seq2[Event, error] {
	if d == nil || d.seq == nil {
		return func(func(Event, error) bool) {}
	}
	return d.seq
}

// Close releases decoder resources.
func (d *SeqDecoder) Close() error {
	if d == nil || d.close == nil {
		return nil
	}
	return d.close()
}

// SourceOf returns the common source value for an event.
func SourceOf(event Event) Source {
	if event == nil {
		return Source{}
	}
	return Source{Driver: event.SourceDriver(), Name: event.SourceName()}
}

// PositionOf returns the common position value for an event.
func PositionOf(event Event) Position {
	if event == nil {
		return Position{}
	}
	return Position{Driver: event.PositionDriver(), Value: event.PositionString()}
}

// CheckpointOf returns a portable checkpoint for the event.
func CheckpointOf(event Event) Checkpoint {
	if event == nil {
		return Checkpoint{}
	}
	return Checkpoint{Source: SourceOf(event), Position: PositionOf(event)}
}

// StartPositionOf returns the checkpoint position carried by open options.
func StartPositionOf(options OpenOptions) Position {
	type startPositionOptions interface {
		StartPosition() Position
	}
	if options == nil {
		return Position{}
	}
	startOptions, ok := options.(startPositionOptions)
	if !ok {
		return Position{}
	}
	return startOptions.StartPosition()
}

// ContextOf returns the context carried by open options, or context.Background.
func ContextOf(options OpenOptions) context.Context {
	type contextOptions interface {
		Context() context.Context
	}
	if options == nil {
		return context.Background()
	}
	ctxOptions, ok := options.(contextOptions)
	if !ok {
		return context.Background()
	}
	ctx := ctxOptions.Context()
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// Bodies filters an event iterator by decoded body type.
func Bodies[T any](seq iter.Seq2[Event, error]) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T
		for event, err := range seq {
			if err != nil {
				yield(zero, err)
				return
			}
			if event == nil {
				continue
			}
			body, ok := event.Body().(T)
			if !ok {
				continue
			}
			if !yield(body, nil) {
				return
			}
		}
	}
}

// Events adapts a typed backend decoder to the common event interface.
func Events[T Event](decoder Decoder[T]) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		if decoder == nil {
			return
		}
		for event, err := range decoder.Events() {
			if !yield(event, err) {
				return
			}
			if err != nil {
				return
			}
		}
	}
}

// Predicate filters common events.
type Predicate func(Event) bool

// Filter yields events accepted by every predicate.
func Filter(seq iter.Seq2[Event, error], predicates ...Predicate) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		for event, err := range seq {
			if err != nil {
				yield(nil, err)
				return
			}
			if event == nil || !match(event, predicates) {
				continue
			}
			if !yield(event, nil) {
				return
			}
		}
	}
}

// ByDriver matches the event source driver.
func ByDriver(driver string) Predicate {
	return func(event Event) bool {
		return event != nil && event.SourceDriver() == driver
	}
}

// BySource matches the event source driver and name.
func BySource(driver, name string) Predicate {
	return func(event Event) bool {
		return event != nil && event.SourceDriver() == driver && event.SourceName() == name
	}
}

// ByKind matches the backend event kind.
func ByKind(kind string) Predicate {
	return func(event Event) bool {
		return event != nil && event.Kind() == kind
	}
}

// Reversible is implemented by events that can emit a compensating operation.
type Reversible interface {
	Reverse() (any, bool)
}

// Flashbacks yields compensating operations from reversible events.
func Flashbacks(seq iter.Seq2[Event, error]) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		for event, err := range seq {
			if err != nil {
				yield(nil, err)
				return
			}
			reversible, ok := event.(Reversible)
			if !ok {
				continue
			}
			op, ok := reversible.Reverse()
			if !ok {
				continue
			}
			if !yield(op, nil) {
				return
			}
		}
	}
}

func match(event Event, predicates []Predicate) bool {
	for _, predicate := range predicates {
		if predicate != nil && !predicate(event) {
			return false
		}
	}
	return true
}

func newOpenOptions(opts ...OpenOption) openOptions {
	var options openOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return options
}
