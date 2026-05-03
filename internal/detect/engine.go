package detect

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hamdyelbatal122/lynxeye/internal/config"
	"github.com/hamdyelbatal122/lynxeye/internal/model"
)

type Engine struct {
	mu       sync.Mutex
	cfg      config.DetectionConfig
	clusters map[string]*model.Cluster
	stats    map[uint64]*patternStats
	nextID   uint64
}

type patternStats struct {
	timestamps []time.Time
	baseline   float64
}

func NewEngine(cfg config.DetectionConfig, persisted []*model.Cluster) *Engine {
	engine := &Engine{
		cfg:      cfg,
		clusters: make(map[string]*model.Cluster, len(persisted)),
		stats:    make(map[uint64]*patternStats, len(persisted)),
	}

	for _, cluster := range persisted {
		copied := *cluster
		engine.clusters[copied.Pattern] = &copied
		engine.stats[copied.ID] = &patternStats{baseline: float64(copied.Count)}
		if copied.ID >= engine.nextID {
			engine.nextID = copied.ID + 1
		}
	}
	if engine.nextID == 0 {
		engine.nextID = 1
	}

	return engine
}

func (e *Engine) Process(event model.Event) model.PatternObservation {
	e.mu.Lock()
	defer e.mu.Unlock()

	normalized := NormalizeMessage(event.Raw)
	cluster, exists := e.clusters[normalized]
	if !exists {
		cluster = &model.Cluster{
			ID:       e.nextID,
			Pattern:  normalized,
			Count:    0,
			LastSeen: event.Timestamp,
			Sample:   event.Raw,
		}
		e.nextID++
		e.clusters[normalized] = cluster
		e.stats[cluster.ID] = &patternStats{}
	}

	cluster.Count++
	cluster.LastSeen = event.Timestamp
	if cluster.Sample == "" {
		cluster.Sample = event.Raw
	}

	stats := e.stats[cluster.ID]
	stats.timestamps = append(stats.timestamps, event.Timestamp)
	stats.timestamps = trimWindow(stats.timestamps, event.Timestamp, e.cfg.Window)

	windowCount := len(stats.timestamps)
	baseline := stats.baseline
	isNewPattern := !exists
	isAnomaly := false
	reason := ""

	if isNewPattern && e.cfg.AlertOnNewPattern {
		reason = "new pattern observed"
	}

	if baseline <= 0 {
		baseline = math.Max(1, float64(windowCount))
	}

	threshold := math.Max(float64(e.cfg.MinEvents), baseline*e.cfg.SpikeMultiplier)
	if !isNewPattern && windowCount >= e.cfg.MinEvents && float64(windowCount) >= threshold {
		isAnomaly = true
		reason = fmt.Sprintf("pattern volume spike: current=%d baseline=%.2f threshold=%.2f", windowCount, baseline, threshold)
	}

	stats.baseline = e.cfg.EWMAAlpha*float64(windowCount) + (1-e.cfg.EWMAAlpha)*baseline

	return model.PatternObservation{
		Event:        event,
		Cluster:      cluster,
		WindowCount:  windowCount,
		Baseline:     baseline,
		IsNewPattern: isNewPattern,
		IsAnomaly:    isAnomaly,
		Reason:       reason,
	}
}

func trimWindow(timestamps []time.Time, now time.Time, window time.Duration) []time.Time {
	cutoff := now.Add(-window)
	trimIndex := 0
	for trimIndex < len(timestamps) && timestamps[trimIndex].Before(cutoff) {
		trimIndex++
	}
	if trimIndex == 0 {
		return timestamps
	}
	remaining := make([]time.Time, len(timestamps)-trimIndex)
	copy(remaining, timestamps[trimIndex:])
	return remaining
}
