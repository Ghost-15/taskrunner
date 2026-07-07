package task

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type fileTasks struct {
	Tasks []rawTask `json:"tasks"`
}

type rawTask struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Params  json.RawMessage `json:"params"`
	Timeout string          `json:"timeout"`
	Retries int             `json:"retries"`
}

func LoadFile(path string) ([]Task, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open tasks file: %w", err)
	}
	defer file.Close()

	tasks, err := Decode(file, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("decode tasks file: %w", err)
	}
	return tasks, nil
}

func Decode(r io.Reader, printWriter io.Writer) ([]Task, error) {
	var payload fileTasks
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	tasks := make([]Task, 0, len(payload.Tasks))
	seen := make(map[string]struct{}, len(payload.Tasks))

	for _, raw := range payload.Tasks {
		task, err := buildTask(raw, printWriter)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[task.ID()]; ok {
			return nil, fmt.Errorf("duplicate task id %q", task.ID())
		}
		seen[task.ID()] = struct{}{}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func buildTask(raw rawTask, printWriter io.Writer) (Task, error) {
	if raw.ID == "" {
		return nil, fmt.Errorf("task id is required")
	}
	if raw.Retries < 0 {
		return nil, fmt.Errorf("task %q has negative retries", raw.ID)
	}

	timeout, err := time.ParseDuration(raw.Timeout)
	if err != nil {
		return nil, fmt.Errorf("task %q invalid timeout: %w", raw.ID, err)
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("task %q timeout must be positive", raw.ID)
	}

	switch raw.Type {
	case "print":
		var params struct {
			Message string `json:"message"`
		}
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		return NewPrintTask(raw.ID, params.Message, timeout, raw.Retries, printWriter), nil
	case "download":
		var params struct {
			URL  string `json:"url"`
			Dest string `json:"dest"`
		}
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		if params.URL == "" || params.Dest == "" {
			return nil, fmt.Errorf("task %q download requires url and dest", raw.ID)
		}
		return NewDownloadTask(raw.ID, params.URL, params.Dest, timeout, raw.Retries), nil
	case "calc":
		var params struct {
			Value int `json:"value"`
		}
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		return NewCalcTask(raw.ID, params.Value, timeout, raw.Retries), nil
	case "fake":
		var params struct {
			Behavior string `json:"behavior"`
			Delay    string `json:"delay"`
		}
		if err := decodeParams(raw, &params); err != nil {
			return nil, err
		}
		behavior, err := parseFakeBehavior(params.Behavior)
		if err != nil {
			return nil, fmt.Errorf("task %q invalid fake behavior: %w", raw.ID, err)
		}
		delay, err := time.ParseDuration(params.Delay)
		if err != nil {
			return nil, fmt.Errorf("task %q invalid fake delay: %w", raw.ID, err)
		}
		return NewConfiguredFakeTask(raw.ID, behavior, delay, timeout, raw.Retries), nil
	default:
		return nil, fmt.Errorf("task %q has unsupported type %q", raw.ID, raw.Type)
	}
}

func decodeParams(raw rawTask, target any) error {
	if len(raw.Params) == 0 {
		return fmt.Errorf("task %q params are required", raw.ID)
	}
	if err := json.Unmarshal(raw.Params, target); err != nil {
		return fmt.Errorf("task %q invalid params: %w", raw.ID, err)
	}
	return nil
}

func parseFakeBehavior(value string) (FakeTaskBehavior, error) {
	switch value {
	case "", "success":
		return BehaviorSuccess, nil
	case "fail":
		return BehaviorFail, nil
	case "timeout":
		return BehaviorTimeout, nil
	default:
		return BehaviorSuccess, fmt.Errorf("unknown behavior %q", value)
	}
}
