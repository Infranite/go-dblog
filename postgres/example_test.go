package postgres_test

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/postgres"
)

func Example_recoveryPlan() {
	var registry dblog.Registry
	if err := postgres.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(postgres.Driver,
		dblog.WithSource(dblog.Source{Name: "slot"}),
		dblog.WithReader(strings.NewReader("table public.users: DELETE: id[integer]:2 name[text]:'Grace'\n")),
	)
	if err != nil {
		panic(err)
	}
	defer closeDecoder(decoder)

	for step, err := range dblog.RecoveryPlan(decoder.Events()) {
		if err != nil {
			panic(err)
		}
		sql := step.Operation.(string)
		fmt.Println(step.Checkpoint.Position.Value, sql)
	}

	// Output:
	// 1 INSERT INTO public.users (id, name) VALUES (2, 'Grace');
}

func closeDecoder(decoder dblog.Decoder[dblog.Event]) {
	if err := decoder.Close(); err != nil {
		panic(err)
	}
}
