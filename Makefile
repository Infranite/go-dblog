GO ?= go
GOLANGCI_LINT ?= golangci-lint

.PHONY: test test-backends test-mysql integration integration-mysql integration-mongo integration-postgres integration-redis lint vet fuzz-smoke bench bench-smoke

test:
	$(GO) test -short ./...
	$(MAKE) test-backends

test-backends:
	cd mysql && GOWORK=off $(GO) test -short ./...
	cd mongo && GOWORK=off $(GO) test -short ./...
	cd postgres && GOWORK=off $(GO) test -short ./...
	cd redis && GOWORK=off $(GO) test -short ./...

test-mysql:
	cd mysql && GOWORK=off $(GO) test -short ./...

integration-mysql:
	./mysql/test/testdata/generate_mysql_binlog.sh mysql:8.4
	./mysql/test/testdata/run_mysql_live.sh mysql:8.4
	cd mysql && GOWORK=off $(GO) test ./...

integration: integration-mysql integration-mongo integration-postgres integration-redis

integration-mongo:
	./mongo/testdata/generate_mongo_oplog.sh mongo:7.0
	./mongo/testdata/run_mongo_live.sh mongo:7.0
	cd mongo && GOWORK=off $(GO) test ./...

integration-postgres:
	./postgres/testdata/generate_postgres_logical.sh postgres:16
	./postgres/testdata/run_postgres_live.sh postgres:16
	cd postgres && GOWORK=off $(GO) test ./...

integration-redis:
	./redis/testdata/generate_redis_aof.sh redis:7.2
	./redis/testdata/run_redis_live.sh redis:7.2
	cd redis && GOWORK=off $(GO) test ./...

lint:
	$(GOLANGCI_LINT) run ./... ./mysql/... ./mongo/... ./postgres/... ./redis/...

vet:
	$(GO) vet ./... ./mysql/... ./mongo/... ./postgres/... ./redis/...

fuzz-smoke:
	cd mysql && GOWORK=off $(GO) test -run '^$$' -fuzz=FuzzDecodeEventHeader -fuzztime=100000x -parallel=2 ./decode/events
	cd mongo && GOWORK=off $(GO) test -run '^$$' -fuzz=FuzzParseLine -fuzztime=100000x -parallel=2 .
	cd postgres && GOWORK=off $(GO) test -run '^$$' -fuzz=FuzzParseLine -fuzztime=100000x -parallel=2 .
	cd redis && GOWORK=off $(GO) test -run '^$$' -fuzz=FuzzParseCommand -fuzztime=100000x -parallel=2 .

bench:
	cd mongo && GOWORK=off $(GO) test -bench=. -benchmem ./...
	cd postgres && GOWORK=off $(GO) test -bench=. -benchmem ./...
	cd redis && GOWORK=off $(GO) test -bench=. -benchmem ./...
	cd mysql && GOWORK=off $(GO) test -bench=. -benchmem ./...

bench-smoke:
	cd mysql && GOWORK=off $(GO) test -run '^$$' -bench=BenchmarkDecoder -benchmem -benchtime=100x ./test
	cd mongo && GOWORK=off $(GO) test -run '^$$' -bench=BenchmarkParseLine -benchmem -benchtime=100x .
	cd postgres && GOWORK=off $(GO) test -run '^$$' -bench=BenchmarkParseLine -benchmem -benchtime=100x .
	cd redis && GOWORK=off $(GO) test -run '^$$' -bench=BenchmarkParseCommand -benchmem -benchtime=100x .
