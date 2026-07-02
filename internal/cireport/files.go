package cireport

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadBenchmarksFromDir reads Go benchmark logs from a downloaded artifact tree.
func LoadBenchmarksFromDir(dir string) ([]BenchmarkResult, error) {
	var results []BenchmarkResult
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".txt" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		module := inferBenchmarkModule(path)
		results = append(results, ParseBenchmarkOutput(module, filepath.Base(path), string(content))...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Module != results[j].Module {
			return results[i].Module < results[j].Module
		}
		if results[i].Name != results[j].Name {
			return results[i].Name < results[j].Name
		}
		return results[i].Source < results[j].Source
	})
	return results, nil
}

// WriteArtifacts writes all CI report files consumed by GitHub Actions.
func WriteArtifacts(dir string, rows []TestMatrixRow, benchmarks []BenchmarkResult) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	files := []struct {
		name    string
		content []byte
	}{
		{name: "tested-matrix.md", content: []byte(renderMatrixMarkdown(rows))},
		{name: "benchmarks.md", content: []byte(renderBenchmarkMarkdown(benchmarks))},
		{name: "ci-report.md", content: []byte(RenderMarkdownReport(rows, benchmarks))},
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(dir, file.name), file.content, 0o644); err != nil {
			return err
		}
	}

	if err := writeJSON(filepath.Join(dir, "tested-matrix.json"), rows); err != nil {
		return err
	}
	if err := writeBenchmarkJSONL(filepath.Join(dir, "benchmarks.jsonl"), benchmarks); err != nil {
		return err
	}
	return nil
}

func renderMatrixMarkdown(rows []TestMatrixRow) string {
	var b strings.Builder
	b.WriteString("## Tested Backend Matrix\n\n")
	b.WriteString("| Product | Runtime | Scope | Job | Result |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, row := range rows {
		fmt.Fprintf(
			&b,
			"| %s | %s | %s | %s | %s |\n",
			escapeMarkdownTable(row.Product),
			escapeMarkdownTable(row.Runtime),
			escapeMarkdownTable(row.Scope),
			escapeMarkdownTable(row.Job),
			escapeMarkdownTable(row.Result),
		)
	}
	return b.String()
}

func renderBenchmarkMarkdown(benchmarks []BenchmarkResult) string {
	var b strings.Builder
	writeBenchmarkMarkdown(&b, benchmarks)
	return b.String()
}

func writeJSON(path string, value any) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeBenchmarkJSONL(path string, benchmarks []BenchmarkResult) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()

	encoder := json.NewEncoder(file)
	for _, benchmark := range benchmarks {
		if err := encoder.Encode(benchmark); err != nil {
			return err
		}
	}
	return nil
}

func inferBenchmarkModule(path string) string {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if module, ok := strings.CutPrefix(stem, "bench-"); ok && module != "" {
		return module
	}

	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if strings.HasPrefix(part, "benchmark-") {
			return strings.TrimPrefix(part, "benchmark-")
		}
	}
	return "unknown"
}
