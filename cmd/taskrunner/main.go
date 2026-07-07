package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"taskrunner/internal/metrics"
	"taskrunner/internal/orchestrator"
	"taskrunner/internal/task"
)

func main() {
	filePath := flag.String("file", "", "path to the tasks JSON file")
	workers := flag.Int("workers", 3, "number of concurrent workers")
	verbose := flag.Bool("verbose", false, "print task status updates to stderr")
	flag.Parse()

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "missing required -file flag")
		os.Exit(2)
	}

	validWorkers, err := orchestrator.ValidateWorkers(*workers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v; falling back to %d workers\n", err, validWorkers)
	}

	tasks, err := task.LoadFile(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load tasks: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	rep, err := orchestrator.Orchestrate(
		ctx,
		tasks,
		validWorkers,
		orchestrator.WithVerbose(*verbose),
		orchestrator.WithStderr(os.Stderr),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "orchestrate: %v\n", err)
	}

	if _, writeErr := rep.WriteTo(os.Stdout); writeErr != nil {
		fmt.Fprintf(os.Stderr, "write report: %v\n", writeErr)
		os.Exit(1)
	}

	if writeErr := os.WriteFile("METRICS.md", []byte(metrics.WriteMetrics(rep.Results)), 0o644); writeErr != nil {
		fmt.Fprintf(os.Stderr, "write metrics: %v\n", writeErr)
		os.Exit(1)
	}

	if err != nil {
		os.Exit(1)
	}
}
