package cireport

import (
	"strings"
	"testing"
)

func TestBuildTestMatrixMapsWorkflowJobs(t *testing.T) {
	jobs := []Job{
		{Name: "root_test", Status: "completed", Conclusion: "success"},
		{Name: "mysql (mysql:8.4)", Status: "completed", Conclusion: "success"},
		{Name: "mongo", Status: "completed", Conclusion: "success"},
		{Name: "postgres", Status: "completed", Conclusion: "success"},
		{Name: "redis", Status: "completed", Conclusion: "success"},
		{Name: "bench (redis)", Status: "completed", Conclusion: "success"},
	}

	rows := BuildTestMatrix(jobs)

	assertMatrixRow(t, rows, TestMatrixRow{
		Product: "Common API",
		Runtime: "Go 1.25.x",
		Scope:   "root package tests, registry tests, checkpoint tests",
		Job:     "root_test",
		Result:  "success",
	})
	assertMatrixRow(t, rows, TestMatrixRow{
		Product: "MySQL",
		Runtime: "mysql:8.4",
		Scope:   "binlog fixture, live replication stream, module tests, decoder benchmark",
		Job:     "mysql (mysql:8.4)",
		Result:  "success",
	})
	assertMatrixRow(t, rows, TestMatrixRow{
		Product: "Redis",
		Runtime: "Go 1.25.x",
		Scope:   "parser benchmark smoke target",
		Job:     "bench (redis)",
		Result:  "success",
	})
}

func TestParseBenchmarkOutput(t *testing.T) {
	const output = `goos: linux
goarch: amd64
pkg: github.com/Infranite/go-dblog/redis
BenchmarkParseCommand-4   	     100	      1135 ns/op	     448 B/op	      10 allocs/op
PASS
`

	results := ParseBenchmarkOutput("redis", "bench-redis.txt", output)

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	got := results[0]
	if got.Module != "redis" {
		t.Fatalf("Module = %q, want redis", got.Module)
	}
	if got.Name != "BenchmarkParseCommand-4" {
		t.Fatalf("Name = %q, want BenchmarkParseCommand-4", got.Name)
	}
	if got.Iterations != 100 {
		t.Fatalf("Iterations = %d, want 100", got.Iterations)
	}
	if got.NsPerOp != 1135 {
		t.Fatalf("NsPerOp = %v, want 1135", got.NsPerOp)
	}
	if got.BytesPerOp != 448 {
		t.Fatalf("BytesPerOp = %v, want 448", got.BytesPerOp)
	}
	if got.AllocsPerOp != 10 {
		t.Fatalf("AllocsPerOp = %v, want 10", got.AllocsPerOp)
	}
}

func TestParseBenchmarkOutputWithThroughput(t *testing.T) {
	const output = `BenchmarkDecoder-4 100 12345 ns/op 0.48 MB/s 27872 B/op 330 allocs/op`

	results := ParseBenchmarkOutput("mysql", "bench-mysql.txt", output)

	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	got := results[0]
	if got.BytesPerOp != 27872 {
		t.Fatalf("BytesPerOp = %v, want 27872", got.BytesPerOp)
	}
	if got.AllocsPerOp != 330 {
		t.Fatalf("AllocsPerOp = %v, want 330", got.AllocsPerOp)
	}
}

func TestRenderReportIncludesMatrixAndBenchmarks(t *testing.T) {
	rows := []TestMatrixRow{{
		Product: "MongoDB",
		Runtime: "mongo:7.0",
		Scope:   "oplog fixture, live change stream, module tests",
		Job:     "mongo",
		Result:  "success",
	}}
	benchmarks := []BenchmarkResult{{
		Module:      "mongo",
		Source:      "bench-mongo.txt",
		Name:        "BenchmarkParseLine-4",
		Iterations:  100,
		NsPerOp:     2100,
		BytesPerOp:  512,
		AllocsPerOp: 8,
	}}

	report := RenderMarkdownReport(rows, benchmarks)

	for _, want := range []string{
		"## Tested Backend Matrix",
		"| MongoDB | mongo:7.0 | oplog fixture, live change stream, module tests | mongo | success |",
		"## Parser Benchmark History",
		"| mongo | BenchmarkParseLine-4 | 100 | 2100 | 512 | 8 |",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("report does not contain %q\n%s", want, report)
		}
	}
}

func assertMatrixRow(t *testing.T, rows []TestMatrixRow, want TestMatrixRow) {
	t.Helper()
	for _, row := range rows {
		if row == want {
			return
		}
	}
	t.Fatalf("missing matrix row %#v in %#v", want, rows)
}
