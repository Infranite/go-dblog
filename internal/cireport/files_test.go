package cireport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBenchmarksFromDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "benchmark-redis", "bench-redis.txt"), `BenchmarkParseCommand-4 100 1135 ns/op 448 B/op 10 allocs/op`)
	writeFile(t, filepath.Join(dir, "benchmark-mongo", "bench-mongo.txt"), `BenchmarkParseLine-4 100 2100 ns/op 512 B/op 8 allocs/op`)
	writeFile(t, filepath.Join(dir, "notes.txt"), `this is not benchmark output`)

	results, err := LoadBenchmarksFromDir(dir)
	if err != nil {
		t.Fatalf("LoadBenchmarksFromDir() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	if results[0].Module != "mongo" || results[1].Module != "redis" {
		t.Fatalf("modules = %q, %q; want mongo, redis", results[0].Module, results[1].Module)
	}
}

func TestWriteArtifacts(t *testing.T) {
	dir := t.TempDir()
	rows := []TestMatrixRow{{
		Product: "Redis",
		Runtime: "redis:7.2",
		Scope:   "AOF fixture, live PSYNC stream, module tests",
		Job:     "redis",
		Result:  "success",
	}}
	benchmarks := []BenchmarkResult{{
		Module:      "redis",
		Source:      "bench-redis.txt",
		Name:        "BenchmarkParseCommand-4",
		Iterations:  100,
		NsPerOp:     1135,
		BytesPerOp:  448,
		AllocsPerOp: 10,
	}}

	if err := WriteArtifacts(dir, rows, benchmarks); err != nil {
		t.Fatalf("WriteArtifacts() error = %v", err)
	}

	for _, name := range []string{
		"tested-matrix.json",
		"tested-matrix.md",
		"benchmarks.jsonl",
		"benchmarks.md",
		"ci-report.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected artifact %s: %v", name, err)
		}
	}

	jsonl := readFile(t, filepath.Join(dir, "benchmarks.jsonl"))
	if !strings.Contains(jsonl, `"module":"redis"`) {
		t.Fatalf("benchmarks.jsonl does not contain redis record: %s", jsonl)
	}
	report := readFile(t, filepath.Join(dir, "ci-report.md"))
	if !strings.Contains(report, "## Tested Backend Matrix") || !strings.Contains(report, "## Parser Benchmark History") {
		t.Fatalf("ci-report.md missing sections:\n%s", report)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	return string(content)
}
