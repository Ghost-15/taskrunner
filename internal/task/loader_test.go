package task

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDecodeBuildsTasksWithSwitch(t *testing.T) {
	t.Parallel()

	input := `{
		"tasks": [
			{"id":"p","type":"print","params":{"message":"hello"},"timeout":"1s","retries":0},
			{"id":"c","type":"calc","params":{"value":10},"timeout":"1s","retries":1},
			{"id":"f","type":"fake","params":{"behavior":"success","delay":"1ms"},"timeout":"1s","retries":0}
		]
	}`

	var printed bytes.Buffer
	tasks, err := Decode(strings.NewReader(input), &printed)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("len(tasks) = %d, want 3", len(tasks))
	}

	for _, task := range tasks {
		if err := task.Execute(context.Background()); err != nil {
			t.Fatalf("Execute(%s) error = %v", task.ID(), err)
		}
	}
	if !strings.Contains(printed.String(), "hello") {
		t.Fatalf("print task wrote %q, want message", printed.String())
	}
}

func TestDecodeRejectsDuplicateIDs(t *testing.T) {
	t.Parallel()

	input := `{
		"tasks": [
			{"id":"same","type":"fake","params":{"behavior":"success","delay":"0s"},"timeout":"1s","retries":0},
			{"id":"same","type":"fake","params":{"behavior":"success","delay":"0s"},"timeout":"1s","retries":0}
		]
	}`

	_, err := Decode(strings.NewReader(input), nil)
	if err == nil {
		t.Fatal("Decode() error = nil, want duplicate id error")
	}
}

func TestFakeTaskReturnsTaskError(t *testing.T) {
	t.Parallel()

	fake := NewConfiguredFakeTask("boom", BehaviorFail, 0, time.Second, 0)
	err := fake.Execute(context.Background())
	if err == nil {
		t.Fatal("Execute() error = nil, want failure")
	}

	var taskErr TaskError
	if !errors.As(err, &taskErr) {
		t.Fatalf("Execute() error = %T, want TaskError", err)
	}
	if taskErr.TaskID != "boom" {
		t.Fatalf("TaskError.TaskID = %q, want boom", taskErr.TaskID)
	}
}
