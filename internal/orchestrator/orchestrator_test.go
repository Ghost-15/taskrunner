package orchestrator

import (
	"bytes"
	"context"
	"testing"
	"time"

	"taskrunner/internal/report"
	"taskrunner/internal/task"
)

func TestValidateWorkers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   int
		want    int
		wantErr bool
	}{
		{name: "valid", input: 4, want: 4},
		{name: "too small", input: 0, want: 3, wantErr: true},
		{name: "too large", input: 101, want: 3, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateWorkers(tt.input)
			if got != tt.want {
				t.Fatalf("ValidateWorkers() = %d, want %d", got, tt.want)
			}
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateWorkers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrchestrateReportsSuccessFailureAndTimeout(t *testing.T) {
	t.Parallel()

	tasks := []task.Task{
		task.NewConfiguredFakeTask("ok", task.BehaviorSuccess, time.Millisecond, 50*time.Millisecond, 0),
		task.NewConfiguredFakeTask("fail", task.BehaviorFail, time.Millisecond, 50*time.Millisecond, 2),
		task.NewConfiguredFakeTask("slow", task.BehaviorSuccess, 30*time.Millisecond, 5*time.Millisecond, 1),
	}

	var stderr bytes.Buffer
	rep, err := Orchestrate(
		context.Background(),
		tasks,
		2,
		WithVerbose(true),
		WithStderr(&stderr),
	)
	if err != nil {
		t.Fatalf("Orchestrate() error = %v", err)
	}

	if len(rep.Results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(rep.Results))
	}

	assertResult(t, rep.Results[0], "ok", report.StatusSuccess, 1)
	assertResult(t, rep.Results[1], "fail", report.StatusFailed, 3)
	assertResult(t, rep.Results[2], "slow", report.StatusTimeout, 2)

	if stderr.Len() == 0 {
		t.Fatal("verbose output is empty")
	}
}

func TestOrchestrateReturnsPartialReportOnCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	tasks := []task.Task{
		task.NewConfiguredFakeTask("running", task.BehaviorTimeout, 0, 100*time.Millisecond, 0),
		task.NewConfiguredFakeTask("queued", task.BehaviorSuccess, time.Millisecond, 100*time.Millisecond, 0),
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	rep, err := Orchestrate(ctx, tasks, 1)
	if err != nil {
		t.Fatalf("Orchestrate() error = %v", err)
	}

	if len(rep.Results) != 1 {
		t.Fatalf("len(results) = %d, want partial report with 1 result", len(rep.Results))
	}
	assertResult(t, rep.Results[0], "running", report.StatusFailed, 1)
}

func assertResult(t *testing.T, got report.TaskResult, id, status string, attempts int) {
	t.Helper()
	if got.ID != id {
		t.Fatalf("result id = %q, want %q", got.ID, id)
	}
	if got.Status != status {
		t.Fatalf("result status for %s = %q, want %q", id, got.Status, status)
	}
	if got.Attempts != attempts {
		t.Fatalf("result attempts for %s = %d, want %d", id, got.Attempts, attempts)
	}
	if got.Duration == "" {
		t.Fatalf("result duration for %s is empty", id)
	}
}
