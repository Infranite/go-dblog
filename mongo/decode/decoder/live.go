package decoder

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo/decode/events/types"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	mongooptions "go.mongodb.org/mongo-driver/v2/mongo/options"
)

type liveWatcher interface {
	Watch(context.Context) (liveCursor, error)
}

type liveCursor interface {
	Next(context.Context) bool
	Decode(any) error
	Err() error
	Close(context.Context) error
}

type driverLiveWatcher struct {
	collection *drivermongo.Collection
}

func (w driverLiveWatcher) Watch(ctx context.Context) (liveCursor, error) {
	return w.collection.Watch(
		ctx,
		drivermongo.Pipeline{},
		mongooptions.ChangeStream().
			SetFullDocument(mongooptions.UpdateLookup).
			SetFullDocumentBeforeChange(mongooptions.WhenAvailable),
	)
}

// LiveDecoder streams MongoDB collection change stream events.
type LiveDecoder struct {
	ctx           context.Context
	source        dblog.Source
	watcher       liveWatcher
	close         func() error
	plugins       []types.EventPlugin
	startPosition int
	position      int
}

// NewLiveDecoder creates a decoder over a live MongoDB collection change stream.
func NewLiveDecoder(
	ctx context.Context,
	source dblog.Source,
	dsn string,
	namespace string,
	opts ...Option,
) (*LiveDecoder, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, types.ErrReaderRequired
	}
	db, collection, err := splitLiveNamespace(namespace)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := drivermongo.Connect(mongooptions.Client().ApplyURI(dsn))
	if err != nil {
		return nil, fmt.Errorf("mongo live connect: %w", err)
	}
	closeClient := func() error {
		return client.Disconnect(context.Background())
	}
	return newLiveDecoder(
		ctx,
		source,
		driverLiveWatcher{collection: client.Database(db).Collection(collection)},
		closeClient,
		namespace,
		opts...,
	), nil
}

func newLiveDecoder(
	ctx context.Context,
	source dblog.Source,
	watcher liveWatcher,
	close func() error,
	namespace string,
	opts ...Option,
) *LiveDecoder {
	if ctx == nil {
		ctx = context.Background()
	}
	if source.Driver == "" {
		source.Driver = types.Driver
	}
	if source.Name == "" {
		source.Name = namespace
	}
	cfg := options{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &LiveDecoder{
		ctx:           ctx,
		source:        source,
		watcher:       watcher,
		close:         close,
		plugins:       cfg.eventPlugins,
		startPosition: cfg.startPosition,
	}
}

func (d *LiveDecoder) Events() iter.Seq2[dblog.Event, error] {
	return func(yield func(dblog.Event, error) bool) {
		if d == nil || d.watcher == nil || d.ctx.Err() != nil {
			return
		}
		cursor, err := d.watcher.Watch(d.ctx)
		if err != nil {
			if d.ctx.Err() == nil && !errors.Is(err, context.Canceled) {
				yield(nil, fmt.Errorf("mongo live watch: %w", err))
			}
			return
		}
		defer func() {
			_ = cursor.Close(context.Background())
		}()

		for cursor.Next(d.ctx) {
			var raw map[string]any
			if err := cursor.Decode(&raw); err != nil {
				yield(nil, fmt.Errorf("mongo live decode: %w", err))
				return
			}
			d.position++
			if d.position <= d.startPosition {
				continue
			}
			event, err := parseRaw(d.source, d.position, nil, raw, d.plugins)
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(event, nil) {
				return
			}
		}
		if err := cursor.Err(); err != nil && d.ctx.Err() == nil {
			yield(nil, fmt.Errorf("mongo live read: %w", err))
		}
	}
}

func (d *LiveDecoder) Close() error {
	if d == nil || d.close == nil {
		return nil
	}
	return d.close()
}

func splitLiveNamespace(namespace string) (string, string, error) {
	db, collection, ok := strings.Cut(strings.TrimSpace(namespace), ".")
	if !ok || db == "" || collection == "" {
		return "", "", types.ErrCollectionRequired
	}
	return db, collection, nil
}

var _ dblog.Decoder[dblog.Event] = (*LiveDecoder)(nil)
