package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"

	"github.com/hamdyelbatal122/lynxeye/internal/config"
	"github.com/hamdyelbatal122/lynxeye/internal/model"
	"github.com/hamdyelbatal122/lynxeye/internal/version"
)

func PrintStartup(cfg config.Config, sourceCount int, notifierCount int, persistedCount int, once bool) {
	title := color.New(color.FgCyan, color.Bold)
	title.Printf("%s %s\n", version.DisplayName, version.Version)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Setting", "Value"})
	table.SetAutoWrapText(false)
	table.Append([]string{"App", cfg.App.Name})
	table.Append([]string{"Sources", fmt.Sprintf("%d", sourceCount)})
	table.Append([]string{"Persisted patterns", fmt.Sprintf("%d", persistedCount)})
	table.Append([]string{"State DB", cfg.State.Path})
	table.Append([]string{"Detection window", cfg.Detection.Window.String()})
	table.Append([]string{"Spike multiplier", fmt.Sprintf("%.2fx", cfg.Detection.SpikeMultiplier)})
	table.Append([]string{"Alerts enabled", fmt.Sprintf("%d notifier(s)", notifierCount)})
	table.Append([]string{"Run mode", ternary(once, "batch", "stream")})
	table.Render()
}

func PrintObservation(observation model.PatternObservation) {
	stamp := observation.Event.Timestamp.Format(time.RFC3339)
	if observation.IsAnomaly {
		color.New(color.FgRed, color.Bold).Printf("[%s] ANOMALY %s | source=%s | pattern=%s\n", stamp, observation.Reason, observation.Event.Source, observation.Cluster.Pattern)
		return
	}
	if observation.IsNewPattern {
		color.New(color.FgYellow).Printf("[%s] NEW PATTERN | source=%s | pattern=%s\n", stamp, observation.Event.Source, observation.Cluster.Pattern)
	}
}

func PrintAlertError(err error) {
	color.New(color.FgHiRed).Printf("alert error: %v\n", err)
}

func PrintShutdown() {
	color.New(color.FgHiBlack).Println("pipeline stopped")
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}
