package mongo_test

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/mongo"
)

func Example_recoveryPlan() {
	var registry dblog.Registry
	if err := mongo.Register(&registry); err != nil {
		panic(err)
	}

	line := strings.Join([]string{
		`{"operationType":"update","ns":{"db":"app","coll":"users"},`,
		`"documentKey":{"_id":1},`,
		`"fullDocument":{"_id":1,"name":"Grace"},`,
		`"fullDocumentBeforeChange":{"_id":1,"name":"Ada"},`,
		`"updateDescription":{"updatedFields":{"name":"Grace"},"removedFields":[]}}`,
	}, "")
	decoder, err := registry.Open(mongo.Driver,
		dblog.WithSource(dblog.Source{Name: "app.users"}),
		dblog.WithReader(strings.NewReader(line+"\n")),
	)
	if err != nil {
		panic(err)
	}
	defer closeDecoder(decoder)

	for step, err := range dblog.RecoveryPlan(decoder.Events()) {
		if err != nil {
			panic(err)
		}
		command := step.Operation.(mongo.Command)
		fmt.Println(step.Checkpoint.Position.Value, command.Operation, command.Document["name"])
	}

	// Output:
	// 1 replace Ada
}

func closeDecoder(decoder dblog.Decoder[dblog.Event]) {
	if err := decoder.Close(); err != nil {
		panic(err)
	}
}
