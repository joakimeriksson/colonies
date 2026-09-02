package cli

import (
	"fmt"
	"strings"

	"github.com/colonyos/colonies/internal/table"
	"github.com/colonyos/colonies/pkg/core"
	"github.com/muesli/termenv"
)

// hours prints a duration in hours, keeping enough precision that a short
// demo run does not show as 0.00.
func hours(seconds float64) string {
	h := seconds / 3600.0
	if h != 0 && h < 0.01 && h > -0.01 {
		return fmt.Sprintf("%.4f", h)
	}
	return fmt.Sprintf("%.2f", h)
}

func signedHours(seconds float64) string {
	if seconds >= 0 {
		return "+" + hours(seconds)
	}
	return hours(seconds)
}

func gbHours(byteSeconds float64) string {
	return fmt.Sprintf("%.2f", byteSeconds/(1024*1024*1024)/3600.0)
}

func displayName(name string, id string) string {
	if name != "" {
		return name
	}
	if id == "" {
		return "colony"
	}
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func printUsageSummaryTable(summaries []*core.UsageSummary, groupBy string) {
	t, theme := createTable(0)

	cols := []table.Column{
		{ID: "key", Name: strings.Title(groupBy), SortIndex: 1},
		{ID: "attempts", Name: "Attempts", SortIndex: 2},
		{ID: "ok", Name: "Success", SortIndex: 3},
		{ID: "failed", Name: "Failed", SortIndex: 4},
		{ID: "running", Name: "Running", SortIndex: 5},
		{ID: "wall", Name: "Wall h", SortIndex: 6},
		{ID: "core", Name: "Core h", SortIndex: 7},
		{ID: "gpu", Name: "GPU h", SortIndex: 8},
		{ID: "mem", Name: "Mem GBh", SortIndex: 9},
		{ID: "mcpu", Name: "Measured CPU h", SortIndex: 10},
	}
	t.SetCols(cols)

	for _, s := range summaries {
		row := []interface{}{
			termenv.String(displayName(s.Name, s.Key)).Foreground(theme.ColorCyan),
			termenv.String(fmt.Sprintf("%d", s.Attempts)).Foreground(theme.ColorGray),
			termenv.String(fmt.Sprintf("%d", s.Succeeded)).Foreground(theme.ColorGreen),
			termenv.String(fmt.Sprintf("%d", s.Failed)).Foreground(theme.ColorRed),
			termenv.String(fmt.Sprintf("%d", s.Running)).Foreground(theme.ColorYellow),
			termenv.String(hours(s.WallSeconds)).Foreground(theme.ColorMagenta),
			termenv.String(hours(s.CoreSeconds)).Foreground(theme.ColorMagenta),
			termenv.String(hours(s.GPUSeconds)).Foreground(theme.ColorMagenta),
			termenv.String(gbHours(s.MemByteSeconds)).Foreground(theme.ColorMagenta),
			termenv.String(hours(s.MeasuredCPUSeconds)).Foreground(theme.ColorViolet),
		}
		t.AddRow(row)
	}

	t.Render()
}

func printUsageRecordsTable(records []*core.UsageRecord) {
	t, theme := createTable(0)

	cols := []table.Column{
		{ID: "process", Name: "Process", SortIndex: 1},
		{ID: "attempt", Name: "Attempt", SortIndex: 2},
		{ID: "func", Name: "Function", SortIndex: 3},
		{ID: "initiator", Name: "Initiator", SortIndex: 4},
		{ID: "executor", Name: "Executor", SortIndex: 5},
		{ID: "provider", Name: "Provider", SortIndex: 6},
		{ID: "start", Name: "Start", SortIndex: 7},
		{ID: "wall", Name: "Wall", SortIndex: 8},
		{ID: "cpu", Name: "CPU", SortIndex: 9},
		{ID: "gpu", Name: "GPU", SortIndex: 10},
		{ID: "outcome", Name: "Outcome", SortIndex: 11},
	}
	t.SetCols(cols)

	for _, r := range records {
		var outcomeColor termenv.Color
		switch r.Outcome {
		case core.USAGE_SUCCESS:
			outcomeColor = theme.ColorGreen
		case core.USAGE_RUNNING:
			outcomeColor = theme.ColorYellow
		default:
			outcomeColor = theme.ColorRed
		}

		row := []interface{}{
			termenv.String(r.ProcessID[:12]).Foreground(theme.ColorGray),
			termenv.String(fmt.Sprintf("%d", r.Attempt)).Foreground(theme.ColorGray),
			termenv.String(r.FuncName).Foreground(theme.ColorCyan),
			termenv.String(displayName(r.InitiatorName, r.InitiatorID)).Foreground(theme.ColorCyan),
			termenv.String(r.ExecutorName).Foreground(theme.ColorBlue),
			termenv.String(displayName(r.ProviderName, r.ProviderID)).Foreground(theme.ColorBlue),
			termenv.String(r.StartTime.Format(TimeLayout)).Foreground(theme.ColorGray),
			termenv.String(r.WallTime().Truncate(1e9).String()).Foreground(theme.ColorMagenta),
			termenv.String(fmt.Sprintf("%dm", r.CPU)).Foreground(theme.ColorMagenta),
			termenv.String(fmt.Sprintf("%d", r.GPUCount)).Foreground(theme.ColorMagenta),
			termenv.String(r.Outcome).Foreground(outcomeColor),
		}
		t.AddRow(row)
	}

	t.Render()
}

func printUsageBalanceTable(balances []*core.UsageBalance) {
	t, theme := createTable(0)

	cols := []table.Column{
		{ID: "member", Name: "Member", SortIndex: 1},
		{ID: "consumed", Name: "Consumed core h", SortIndex: 2},
		{ID: "provided", Name: "Provided core h", SortIndex: 3},
		{ID: "self", Name: "Self core h", SortIndex: 4},
		{ID: "net", Name: "Net core h", SortIndex: 5},
		{ID: "gconsumed", Name: "Consumed GPU h", SortIndex: 6},
		{ID: "gprovided", Name: "Provided GPU h", SortIndex: 7},
		{ID: "gnet", Name: "Net GPU h", SortIndex: 8},
	}
	t.SetCols(cols)

	for _, b := range balances {
		netColor := theme.ColorGreen
		if b.NetCoreSeconds() < 0 {
			netColor = theme.ColorRed
		}
		gpuNetColor := theme.ColorGreen
		if b.NetGPUSeconds() < 0 {
			gpuNetColor = theme.ColorRed
		}

		row := []interface{}{
			termenv.String(displayName(b.UserName, b.UserID)).Foreground(theme.ColorCyan),
			termenv.String(hours(b.ConsumedCoreSeconds)).Foreground(theme.ColorMagenta),
			termenv.String(hours(b.ProvidedCoreSeconds)).Foreground(theme.ColorMagenta),
			termenv.String(hours(b.SelfCoreSeconds)).Foreground(theme.ColorGray),
			termenv.String(signedHours(b.NetCoreSeconds())).Foreground(netColor),
			termenv.String(hours(b.ConsumedGPUSeconds)).Foreground(theme.ColorMagenta),
			termenv.String(hours(b.ProvidedGPUSeconds)).Foreground(theme.ColorMagenta),
			termenv.String(signedHours(b.NetGPUSeconds())).Foreground(gpuNetColor),
		}
		t.AddRow(row)
	}

	t.Render()
}
