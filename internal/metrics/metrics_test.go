package metrics

import (
	"strings"
	"testing"

	"taskrunner/internal/report"
)

func TestWriteMetricsCountsStatuses(t *testing.T) {
	t.Parallel()

	content := WriteMetrics([]report.TaskResult{
		{ID: "ok", Status: report.StatusSuccess},
		{ID: "fail", Status: report.StatusFailed},
		{ID: "slow", Status: report.StatusTimeout},
	})

	for _, want := range []string{
		"- Taches executees : 3",
		"- Taches reussies : 1",
		"- Taches en echec : 1",
		"- Taches en timeout : 1",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("metrics missing %q in:\n%s", want, content)
		}
	}
}
