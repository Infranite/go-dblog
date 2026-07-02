package redis_test

import (
	"fmt"
	"strings"

	"github.com/Infranite/go-dblog"
	"github.com/Infranite/go-dblog/redis"
)

func ExampleRegister() {
	var registry dblog.Registry
	if err := redis.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(redis.Driver,
		dblog.WithReader(strings.NewReader("*3\r\n$4\r\nSADD\r\n$4\r\ntags\r\n$2\r\ngo\r\n")),
	)
	if err != nil {
		panic(err)
	}
	defer closeDecoder(decoder)

	for event, err := range dblog.Filter(decoder.Events(), dblog.ByKind(redis.CommandSAdd)) {
		if err != nil {
			panic(err)
		}
		command := event.Body().(redis.Command)
		fmt.Println(event.Kind(), command.Args)
	}

	// Output:
	// sadd [tags go]
}

func Example_flashback() {
	var registry dblog.Registry
	if err := redis.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(redis.Driver,
		dblog.WithReader(strings.NewReader("*2\r\n$4\r\nINCR\r\n$7\r\ncounter\r\n")),
	)
	if err != nil {
		panic(err)
	}
	defer closeDecoder(decoder)

	for op, err := range dblog.Flashbacks(decoder.Events()) {
		if err != nil {
			panic(err)
		}
		command := op.(redis.Command)
		fmt.Println(command.Name, command.Args)
	}

	// Output:
	// decr [counter]
}

func Example_recoveryPlan() {
	var registry dblog.Registry
	if err := redis.Register(&registry); err != nil {
		panic(err)
	}

	decoder, err := registry.Open(redis.Driver,
		dblog.WithSource(dblog.Source{Name: "appendonly.aof"}),
		dblog.WithReader(strings.NewReader("*2\r\n$4\r\nINCR\r\n$7\r\ncounter\r\n")),
	)
	if err != nil {
		panic(err)
	}
	defer closeDecoder(decoder)

	for step, err := range dblog.RecoveryPlan(decoder.Events()) {
		if err != nil {
			panic(err)
		}
		command := step.Operation.(redis.Command)
		fmt.Println(step.Checkpoint.Position.Value, command.Name, command.Args)
	}

	// Output:
	// 1 decr [counter]
}

func closeDecoder(decoder dblog.Decoder[dblog.Event]) {
	if err := decoder.Close(); err != nil {
		panic(err)
	}
}
