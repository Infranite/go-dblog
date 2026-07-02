package cireport

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Job is the subset of a GitHub Actions job payload used by the CI report.
type Job struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// JobsPayload is the GitHub Actions jobs API response shape.
type JobsPayload struct {
	Jobs []Job `json:"jobs"`
}

// TestMatrixRow describes one expected CI coverage row and its observed result.
type TestMatrixRow struct {
	Product string `json:"product"`
	Runtime string `json:"runtime"`
	Scope   string `json:"scope"`
	Job     string `json:"job"`
	Result  string `json:"result"`
}

// BenchmarkResult is one parsed Go benchmark line.
type BenchmarkResult struct {
	Module      string  `json:"module"`
	Source      string  `json:"source"`
	Name        string  `json:"name"`
	Iterations  int     `json:"iterations"`
	NsPerOp     float64 `json:"ns_per_op"`
	BytesPerOp  float64 `json:"bytes_per_op,omitempty"`
	AllocsPerOp float64 `json:"allocs_per_op,omitempty"`
}

type expectedJob struct {
	product string
	runtime string
	scope   string
	name    string
}

var expectedJobs = []expectedJob{
	{
		product: "Common API",
		runtime: "Go 1.25.x",
		scope:   "root package tests, registry tests, checkpoint tests",
		name:    "root_test",
	},
	{
		product: "MySQL",
		runtime: "Go 1.25.x",
		scope:   "short race tests",
		name:    "backend_short (mysql)",
	},
	{
		product: "MongoDB",
		runtime: "Go 1.25.x",
		scope:   "short race tests",
		name:    "backend_short (mongo)",
	},
	{
		product: "PostgreSQL",
		runtime: "Go 1.25.x",
		scope:   "short race tests",
		name:    "backend_short (postgres)",
	},
	{
		product: "Redis",
		runtime: "Go 1.25.x",
		scope:   "short race tests",
		name:    "backend_short (redis)",
	},
	{
		product: "MySQL",
		runtime: "mysql:5.6",
		scope:   "binlog fixture and module tests",
		name:    "mysql (mysql:5.6)",
	},
	{
		product: "MySQL",
		runtime: "mysql:5.7",
		scope:   "binlog fixture and module tests",
		name:    "mysql (mysql:5.7)",
	},
	{
		product: "MySQL",
		runtime: "mysql:8.0",
		scope:   "binlog fixture and module tests",
		name:    "mysql (mysql:8.0)",
	},
	{
		product: "MySQL",
		runtime: "mysql:8.4",
		scope:   "binlog fixture, live replication stream, module tests, decoder benchmark",
		name:    "mysql (mysql:8.4)",
	},
	{
		product: "MongoDB",
		runtime: "mongo:7.0",
		scope:   "oplog fixture, live change stream, module tests",
		name:    "mongo",
	},
	{
		product: "PostgreSQL",
		runtime: "postgres:16",
		scope:   "logical decoding fixture, live SQL and wire readers, module tests",
		name:    "postgres",
	},
	{
		product: "Redis",
		runtime: "redis:7.2",
		scope:   "AOF fixture, live PSYNC stream, module tests",
		name:    "redis",
	},
	{
		product: "MySQL",
		runtime: "Go 1.25.x",
		scope:   "parser fuzz smoke target",
		name:    "fuzz (mysql)",
	},
	{
		product: "MongoDB",
		runtime: "Go 1.25.x",
		scope:   "parser fuzz smoke target",
		name:    "fuzz (mongo)",
	},
	{
		product: "PostgreSQL",
		runtime: "Go 1.25.x",
		scope:   "parser fuzz smoke target",
		name:    "fuzz (postgres)",
	},
	{
		product: "Redis",
		runtime: "Go 1.25.x",
		scope:   "parser fuzz smoke target",
		name:    "fuzz (redis)",
	},
	{
		product: "MySQL",
		runtime: "Go 1.25.x",
		scope:   "parser benchmark smoke target",
		name:    "bench (mysql)",
	},
	{
		product: "MongoDB",
		runtime: "Go 1.25.x",
		scope:   "parser benchmark smoke target",
		name:    "bench (mongo)",
	},
	{
		product: "PostgreSQL",
		runtime: "Go 1.25.x",
		scope:   "parser benchmark smoke target",
		name:    "bench (postgres)",
	},
	{
		product: "Redis",
		runtime: "Go 1.25.x",
		scope:   "parser benchmark smoke target",
		name:    "bench (redis)",
	},
}

// BuildTestMatrix combines the expected CI matrix with observed GitHub job results.
func BuildTestMatrix(jobs []Job) []TestMatrixRow {
	results := make(map[string]string, len(jobs))
	for _, job := range jobs {
		results[job.Name] = resultOf(job)
	}

	rows := make([]TestMatrixRow, 0, len(expectedJobs))
	for _, expected := range expectedJobs {
		result, ok := results[expected.name]
		if !ok {
			result = "missing"
		}
		rows = append(rows, TestMatrixRow{
			Product: expected.product,
			Runtime: expected.runtime,
			Scope:   expected.scope,
			Job:     expected.name,
			Result:  result,
		})
	}
	return rows
}

// ParseBenchmarkOutput extracts Go benchmark metrics from a benchmark command log.
func ParseBenchmarkOutput(module, source, output string) []BenchmarkResult {
	lines := strings.Split(output, "\n")
	results := make([]BenchmarkResult, 0, 1)
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.HasPrefix(fields[0], "Benchmark") || fields[3] != "ns/op" {
			continue
		}

		iterations, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		nsPerOp, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			continue
		}

		result := BenchmarkResult{
			Module:     module,
			Source:     source,
			Name:       fields[0],
			Iterations: iterations,
			NsPerOp:    nsPerOp,
		}
		for i := 4; i < len(fields)-1; i++ {
			switch fields[i+1] {
			case "B/op":
				if bytesPerOp, err := strconv.ParseFloat(fields[i], 64); err == nil {
					result.BytesPerOp = bytesPerOp
				}
			case "allocs/op":
				if allocsPerOp, err := strconv.ParseFloat(fields[i], 64); err == nil {
					result.AllocsPerOp = allocsPerOp
				}
			}
		}
		results = append(results, result)
	}
	return results
}

// RenderMarkdownReport creates the workflow summary and artifact report body.
func RenderMarkdownReport(rows []TestMatrixRow, benchmarks []BenchmarkResult) string {
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

	b.WriteString("\n")
	writeBenchmarkMarkdown(&b, benchmarks)
	return b.String()
}

func writeBenchmarkMarkdown(b *strings.Builder, benchmarks []BenchmarkResult) {
	b.WriteString("## Parser Benchmark History\n\n")
	if len(benchmarks) == 0 {
		b.WriteString("No parser benchmark output was found.\n")
		return
	}

	sorted := append([]BenchmarkResult(nil), benchmarks...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Module != sorted[j].Module {
			return sorted[i].Module < sorted[j].Module
		}
		return sorted[i].Name < sorted[j].Name
	})

	b.WriteString("| Module | Benchmark | Iterations | ns/op | B/op | allocs/op |\n")
	b.WriteString("|---|---|---:|---:|---:|---:|\n")
	for _, benchmark := range sorted {
		fmt.Fprintf(
			b,
			"| %s | %s | %d | %s | %s | %s |\n",
			escapeMarkdownTable(benchmark.Module),
			escapeMarkdownTable(benchmark.Name),
			benchmark.Iterations,
			formatFloat(benchmark.NsPerOp),
			formatFloat(benchmark.BytesPerOp),
			formatFloat(benchmark.AllocsPerOp),
		)
	}
}

func resultOf(job Job) string {
	if job.Conclusion != "" {
		return job.Conclusion
	}
	if job.Status != "" {
		return job.Status
	}
	return "unknown"
}

func escapeMarkdownTable(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

func formatFloat(value float64) string {
	if value == 0 {
		return "0"
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}
