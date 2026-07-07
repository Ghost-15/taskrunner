package task

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	ErrCodeExecution = 1
	ErrCodeHTTP      = 2
	ErrCodeIO        = 3
	ErrCodeCanceled  = 4
)

type Task interface {
	ID() string
	Execute(ctx context.Context) error
}

type TaskError struct {
	Code   int
	TaskID string
	Err    error
}

func (e TaskError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("task %s failed with code %d", e.TaskID, e.Code)
	}
	return fmt.Sprintf("task %s failed with code %d: %v", e.TaskID, e.Code, e.Err)
}

func (e TaskError) Unwrap() error {
	return e.Err
}

type Metadata interface {
	Timeout() time.Duration
	Retries() int
}

type BaseTask struct {
	id      string
	timeout time.Duration
	retries int
}

func NewBaseTask(id string, timeout time.Duration, retries int) BaseTask {
	return BaseTask{id: id, timeout: timeout, retries: retries}
}

func (t BaseTask) ID() string {
	return t.id
}

func (t BaseTask) Timeout() time.Duration {
	return t.timeout
}

func (t BaseTask) Retries() int {
	return t.retries
}

type PrintTask struct {
	BaseTask
	message string
	writer  io.Writer
}

func NewPrintTask(id, message string, timeout time.Duration, retries int, writer io.Writer) *PrintTask {
	if writer == nil {
		writer = os.Stderr
	}
	return &PrintTask{
		BaseTask: NewBaseTask(id, timeout, retries),
		message:  message,
		writer:   writer,
	}
}

func (t *PrintTask) Execute(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return TaskError{Code: ErrCodeCanceled, TaskID: t.ID(), Err: ctx.Err()}
	default:
	}

	if _, err := fmt.Fprintln(t.writer, t.message); err != nil {
		return TaskError{Code: ErrCodeIO, TaskID: t.ID(), Err: err}
	}
	return nil
}

type DownloadTask struct {
	BaseTask
	url  string
	dest string
}

func NewDownloadTask(id, url, dest string, timeout time.Duration, retries int) *DownloadTask {
	return &DownloadTask{
		BaseTask: NewBaseTask(id, timeout, retries),
		url:      url,
		dest:     dest,
	}
}

func (t *DownloadTask) Execute(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url, nil)
	if err != nil {
		return TaskError{Code: ErrCodeExecution, TaskID: t.ID(), Err: err}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TaskError{Code: ErrCodeHTTP, TaskID: t.ID(), Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return TaskError{
			Code:   ErrCodeHTTP,
			TaskID: t.ID(),
			Err:    fmt.Errorf("unexpected status %s", resp.Status),
		}
	}

	if err := os.MkdirAll(filepath.Dir(t.dest), 0o755); err != nil {
		return TaskError{Code: ErrCodeIO, TaskID: t.ID(), Err: err}
	}

	file, err := os.Create(t.dest)
	if err != nil {
		return TaskError{Code: ErrCodeIO, TaskID: t.ID(), Err: err}
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return TaskError{Code: ErrCodeIO, TaskID: t.ID(), Err: err}
	}
	return nil
}

type CalcTask struct {
	BaseTask
	value int
}

func NewCalcTask(id string, value int, timeout time.Duration, retries int) *CalcTask {
	return &CalcTask{
		BaseTask: NewBaseTask(id, timeout, retries),
		value:    value,
	}
}

func (t *CalcTask) Execute(ctx context.Context) error {
	limit := max(1, t.value)
	result := 0.0

	for i := 1; i <= limit; i++ {
		if i%1000 == 0 {
			select {
			case <-ctx.Done():
				return TaskError{Code: ErrCodeCanceled, TaskID: t.ID(), Err: ctx.Err()}
			default:
			}
		}
		result += math.Sqrt(float64(i))
	}

	if math.IsNaN(result) {
		return TaskError{Code: ErrCodeExecution, TaskID: t.ID(), Err: errors.New("calculation produced NaN")}
	}
	return nil
}

type FakeTaskBehavior int

const (
	BehaviorSuccess FakeTaskBehavior = iota
	BehaviorFail
	BehaviorTimeout
)

type FakeTask struct {
	BaseTask
	behavior FakeTaskBehavior
	delay    time.Duration
}

func NewFakeTask(id string, behavior FakeTaskBehavior, delay time.Duration) *FakeTask {
	return &FakeTask{
		BaseTask: NewBaseTask(id, time.Second, 0),
		behavior: behavior,
		delay:    delay,
	}
}

func NewConfiguredFakeTask(id string, behavior FakeTaskBehavior, delay, timeout time.Duration, retries int) *FakeTask {
	return &FakeTask{
		BaseTask: NewBaseTask(id, timeout, retries),
		behavior: behavior,
		delay:    delay,
	}
}

func (t *FakeTask) Execute(ctx context.Context) error {
	if t.delay > 0 {
		select {
		case <-time.After(t.delay):
		case <-ctx.Done():
			return TaskError{Code: ErrCodeCanceled, TaskID: t.ID(), Err: ctx.Err()}
		}
	}

	switch t.behavior {
	case BehaviorSuccess:
		return nil
	case BehaviorFail:
		return TaskError{Code: ErrCodeExecution, TaskID: t.ID(), Err: errors.New("fake task failed")}
	case BehaviorTimeout:
		<-ctx.Done()
		return TaskError{Code: ErrCodeCanceled, TaskID: t.ID(), Err: ctx.Err()}
	default:
		return TaskError{Code: ErrCodeExecution, TaskID: t.ID(), Err: fmt.Errorf("unknown fake behavior %d", t.behavior)}
	}
}
