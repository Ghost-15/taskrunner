package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"taskrunner/internal/report"
	"taskrunner/internal/task"
)

type Option func(*OrchestratorConfig)

type OrchestratorConfig struct {
	Workers int
	Verbose bool
	Stderr  io.Writer
}

func WithWorkers(n int) Option {
	return func(c *OrchestratorConfig) {
		c.Workers = n
	}
}

func WithVerbose(v bool) Option {
	return func(c *OrchestratorConfig) {
		c.Verbose = v
	}
}

func WithStderr(w io.Writer) Option {
	return func(c *OrchestratorConfig) {
		c.Stderr = w
	}
}

func ValidateWorkers(n int) (int, error) {
	if n < 1 || n > 100 {
		return 3, fmt.Errorf("workers must be between 1 and 100, got %d", n)
	}
	return n, nil
}

func Orchestrate(ctx context.Context, tasks []task.Task, workers int, opts ...Option) (report.Report, error) {
	opts = append([]Option{WithWorkers(workers)}, opts...)
	cfg := OrchestratorConfig{
		Workers: 3,
		Stderr:  os.Stderr,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	validWorkers, err := ValidateWorkers(cfg.Workers)
	if err != nil {
		cfg.Workers = validWorkers
		return report.Report{}, fmt.Errorf("validate workers: %w", err)
	}
	cfg.Workers = validWorkers
	if cfg.Stderr == nil {
		cfg.Stderr = io.Discard
	}

	jobs := make(chan job)
	results := make(chan indexedResult)
	var wg sync.WaitGroup

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go worker(ctx, jobs, results, cfg, &wg)
	}

	go func() {
		defer close(jobs)
		for i, t := range tasks {
			select {
			case <-ctx.Done():
				return
			case jobs <- job{index: i, task: t}:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	completed := make([]indexedResult, 0, len(tasks))
	for result := range results {
		completed = append(completed, result)
	}

	sort.SliceStable(completed, func(i, j int) bool {
		return completed[i].index < completed[j].index
	})

	rep := report.Report{Results: make([]report.TaskResult, 0, len(completed))}
	for _, result := range completed {
		rep.Results = append(rep.Results, result.result)
	}

	return rep, nil
}

type job struct {
	index int
	task  task.Task
}

type indexedResult struct {
	index  int
	result report.TaskResult
}

func worker(ctx context.Context, jobs <-chan job, results chan<- indexedResult, cfg OrchestratorConfig, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case j, ok := <-jobs:
			if !ok {
				return
			}
			results <- indexedResult{index: j.index, result: runTask(ctx, j.task, cfg)}
		}
	}
}

func runTask(ctx context.Context, t task.Task, cfg OrchestratorConfig) report.TaskResult {
	timeout, retries := taskExecutionPolicy(t)
	maxAttempts := retries + 1
	status := report.StatusFailed
	started := time.Now()
	attempts := 0

	for attempts < maxAttempts {
		attempts++
		logVerbose(cfg, "start", t.ID(), attempts)

		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		err := t.Execute(attemptCtx)
		attemptErr := attemptCtx.Err()
		cancel()

		if err == nil {
			status = report.StatusSuccess
			logVerbose(cfg, "success", t.ID(), attempts)
			break
		}

		if errors.Is(err, context.DeadlineExceeded) || errors.Is(attemptErr, context.DeadlineExceeded) {
			status = report.StatusTimeout
			logVerbose(cfg, "timeout", t.ID(), attempts)
		} else {
			status = report.StatusFailed
			logVerbose(cfg, "failed", t.ID(), attempts)
		}

		if ctx.Err() != nil && !errors.Is(attemptErr, context.DeadlineExceeded) {
			break
		}
	}

	return report.TaskResult{
		ID:       t.ID(),
		Status:   status,
		Duration: time.Since(started).String(),
		Attempts: attempts,
	}
}

func taskExecutionPolicy(t task.Task) (time.Duration, int) {
	metadata, ok := t.(task.Metadata)
	if !ok {
		return time.Second, 0
	}
	return metadata.Timeout(), metadata.Retries()
}

func logVerbose(cfg OrchestratorConfig, status, id string, attempt int) {
	if !cfg.Verbose {
		return
	}
	fmt.Fprintf(cfg.Stderr, "[%s] task=%s attempt=%d\n", status, id, attempt)
}
