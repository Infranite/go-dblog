package mongo

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Infranite/go-dblog"
	drivermongo "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const mongoFixturePath = "testdata/oplog.jsonl"
const mongoLiveTimeout = 15 * time.Second

func TestFixtureBackedOplogDecoding(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fixture-backed integration test in short mode")
	}
	path := requireMongoFixture(t)

	decoder := openMongoFixture(t, path)
	counts := map[string]int{}
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		change := event.Body().(Change)
		if change.Database != "dblog_ci" || change.Collection != "users" {
			t.Fatalf("change namespace = %#v", change)
		}
		counts[event.Kind()]++
	}

	for _, kind := range []string{OperationInsert, OperationUpdate, OperationDelete} {
		if counts[kind] == 0 {
			t.Fatalf("fixture has no %s events: %v", kind, counts)
		}
	}

	decoder = openMongoFixture(t, path)
	var inserts int
	for event, err := range dblog.Filter(decoder.Events(), dblog.ByKind(OperationInsert)) {
		if err != nil {
			t.Fatal(err)
		}
		if event.Kind() != OperationInsert {
			t.Fatalf("filtered event kind = %s", event.Kind())
		}
		inserts++
	}
	if inserts == 0 {
		t.Fatal("filtered insert events are empty")
	}

	decoder = openMongoFixture(t, path)
	var flashbacks int
	for op, err := range dblog.Flashbacks(decoder.Events()) {
		if err != nil {
			t.Fatal(err)
		}
		command := op.(Command)
		if command.Database != "dblog_ci" || command.Collection != "users" {
			t.Fatalf("flashback command = %#v", command)
		}
		flashbacks++
	}
	if flashbacks == 0 {
		t.Fatal("fixture produced no flashback commands")
	}
}

func TestLiveChangeStream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live integration test in short mode")
	}
	dsn := os.Getenv("DBLOG_MONGO_LIVE_DSN")
	if dsn == "" {
		t.Skip("set DBLOG_MONGO_LIVE_DSN to run live test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), mongoLiveTimeout)
	defer cancel()

	client, err := drivermongo.Connect(options.Client().ApplyURI(dsn))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Disconnect(context.Background()); err != nil {
			t.Fatal(err)
		}
	})

	database := client.Database("dblog_ci")
	_ = database.Collection("users").Drop(ctx)
	if err := database.CreateCollection(ctx, "users"); err != nil {
		t.Fatal(err)
	}
	collection := database.Collection("users")

	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}
	decoder, err := registry.Open(Driver,
		dblog.WithContext(ctx),
		dblog.WithDSN(dsn),
		dblog.WithSource(dblog.Source{Name: "dblog_ci.users"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := decoder.Close(); err != nil {
			t.Fatal(err)
		}
	})

	writeErr := make(chan error, 1)
	go func() {
		time.Sleep(500 * time.Millisecond)
		if _, err := collection.InsertOne(ctx, map[string]any{"_id": 1, "name": "Ada"}); err != nil {
			writeErr <- err
			return
		}
		if _, err := collection.UpdateOne(ctx, map[string]any{"_id": 1}, map[string]any{"$set": map[string]any{"name": "Ada Lovelace"}}); err != nil {
			writeErr <- err
			return
		}
		if _, err := collection.DeleteOne(ctx, map[string]any{"_id": 1}); err != nil {
			writeErr <- err
			return
		}
		writeErr <- nil
	}()

	counts := map[string]int{}
	for event, err := range decoder.Events() {
		if err != nil {
			t.Fatal(err)
		}
		change := event.Body().(Change)
		if change.Database != "dblog_ci" || change.Collection != "users" {
			t.Fatalf("live change namespace = %#v", change)
		}
		counts[event.Kind()]++
		if counts[OperationInsert] > 0 && counts[OperationUpdate] > 0 && counts[OperationDelete] > 0 {
			cancel()
		}
	}
	if err := <-writeErr; err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{OperationInsert, OperationUpdate, OperationDelete} {
		if counts[kind] == 0 {
			t.Fatalf("live reader has no %s events: %v", kind, counts)
		}
	}
}

func requireMongoFixture(t *testing.T) string {
	t.Helper()
	if _, err := os.Stat(mongoFixturePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("test fixture %s not found; run testdata/generate_mongo_oplog.sh", mongoFixturePath)
		}
		t.Fatal(err)
	}
	return mongoFixturePath
}

func openMongoFixture(t *testing.T, path string) dblog.Decoder[dblog.Event] {
	t.Helper()
	var registry dblog.Registry
	if err := Register(&registry); err != nil {
		t.Fatal(err)
	}
	decoder, err := registry.Open(Driver,
		dblog.WithPath(path),
		dblog.WithSource(dblog.Source{Name: "oplog"}),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := decoder.Close(); err != nil {
			t.Fatal(err)
		}
	})
	return decoder
}
