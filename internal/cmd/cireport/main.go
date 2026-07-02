package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Infranite/go-dblog/internal/cireport"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("cireport", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var jobsPath string
	var benchDir string
	var outDir string
	flags.StringVar(&jobsPath, "jobs", "", "path to GitHub Actions jobs JSON")
	flags.StringVar(&benchDir, "bench-dir", "", "directory containing benchmark artifacts")
	flags.StringVar(&outDir, "out", ".ci-report", "output directory")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if jobsPath == "" {
		return fmt.Errorf("-jobs is required")
	}
	if benchDir == "" {
		return fmt.Errorf("-bench-dir is required")
	}

	payload, err := readJobs(jobsPath)
	if err != nil {
		return err
	}
	benchmarks, err := cireport.LoadBenchmarksFromDir(benchDir)
	if err != nil {
		return err
	}
	rows := cireport.BuildTestMatrix(payload.Jobs)
	if err := cireport.WriteArtifacts(outDir, rows, benchmarks); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(stdout, "wrote CI report to %s\n", outDir); err != nil {
		return err
	}
	return nil
}

func readJobs(path string) (cireport.JobsPayload, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return cireport.JobsPayload{}, err
	}

	var payload cireport.JobsPayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return cireport.JobsPayload{}, err
	}
	return payload, nil
}
