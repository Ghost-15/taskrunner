package metrics

import (
	"fmt"
	"runtime"
	"strings"

	"taskrunner/internal/report"
)

func WriteMetrics(results []report.TaskResult) string {
	var success, failed, timeout int
	for _, result := range results {
		switch result.Status {
		case report.StatusSuccess:
			success++
		case report.StatusFailed:
			failed++
		case report.StatusTimeout:
			timeout++
		}
	}

	var b strings.Builder
	fmt.Fprintln(&b, "# Metriques d'execution")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "- Goroutines actives a la fin : %d\n", runtime.NumGoroutine())
	fmt.Fprintf(&b, "- Taches executees : %d\n", len(results))
	fmt.Fprintf(&b, "- Taches reussies : %d\n", success)
	fmt.Fprintf(&b, "- Taches en echec : %d\n", failed)
	fmt.Fprintf(&b, "- Taches en timeout : %d\n", timeout)
	return b.String()
}
