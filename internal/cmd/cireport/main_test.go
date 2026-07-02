package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunWritesReportArtifacts(t *testing.T) {
	dir := t.TempDir()
	jobsPath := filepath.Join(dir, "jobs.json")
	benchDir := filepath.Join(dir, "benchmarks")
	outDir := filepath.Join(dir, "out")

	writeFile(t, jobsPath, `{
  "jobs": [
    {"name": "root_test", "status": "completed", "conclusion": "success"},
    {"name": "redis", "status": "completed", "conclusion": "success"},
    {"name": "bench (redis)", "status": "completed", "conclusion": "success"}
  ]
}`)
	writeFile(t, filepath.Join(benchDir, "benchmark-redis", "bench-redis.txt"), `BenchmarkParseCommand-4 100 1135 ns/op 448 B/op 10 allocs/op`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{
		"-jobs", jobsPath,
		"-bench-dir", benchDir,
		"-out", outDir,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run() error = %v; stderr=%s", err, stderr.String())
	}

	report, err := os.ReadFile(filepath.Join(outDir, "ci-report.md"))
	if err != nil {
		t.Fatalf("ReadFile(ci-report.md) error = %v", err)
	}
	if !strings.Contains(string(report), "| Redis | redis:7.2 | AOF fixture, live PSYNC stream, module tests | redis | success |") {
		t.Fatalf("ci-report.md missing redis matrix row:\n%s", string(report))
	}
	if !strings.Contains(stdout.String(), "wrote CI report") {
		t.Fatalf("stdout = %q, want success message", stdout.String())
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
