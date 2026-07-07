package report

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestReportWriteToWritesJSON(t *testing.T) {
	t.Parallel()

	rep := Report{Results: []TaskResult{
		{ID: "t1", Status: StatusSuccess, Duration: "1ms", Attempts: 1},
	}}

	var out bytes.Buffer
	n, err := rep.WriteTo(&out)
	if err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}
	if n != int64(out.Len()) {
		t.Fatalf("WriteTo() n = %d, want %d", n, out.Len())
	}

	var decoded Report
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if decoded.Results[0].ID != "t1" {
		t.Fatalf("decoded id = %q, want t1", decoded.Results[0].ID)
	}
}
