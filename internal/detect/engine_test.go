package detect

import (
	"testing"
	"time"

	"github.com/hamdyelbatal122/lynxeye/internal/config"
	"github.com/hamdyelbatal122/lynxeye/internal/model"
)

func TestNormalizeMessageCollapsesVolatileTokens(t *testing.T) {
	message := "2026-05-03T11:20:00Z request_id=123e4567-e89b-12d3-a456-426614174000 user=42 ip=10.0.0.9"
	got := NormalizeMessage(message)

	want := "2026-05-03t11:20:00z request_id=123e4567-e89b-12d3-a456-426614174000 user=42 ip=10.0.0.9"
	if got == want {
		t.Fatalf("normalization did not collapse dynamic tokens: %q", got)
	}

	if got == "<empty>" {
		t.Fatalf("normalization unexpectedly returned empty placeholder")
	}
}

func TestEngineDetectsSpikeAgainstBaseline(t *testing.T) {
	cfg := config.DetectionConfig{
		Window:            2 * time.Minute,
		MinEvents:         3,
		SpikeMultiplier:   1.2,
		EWMAAlpha:         0.5,
		AlertOnNewPattern: true,
	}
	engine := NewEngine(cfg, nil)
	baseTime := time.Now()

	first := engine.Process(model.Event{Source: "unit", Raw: "cache miss for user 10", Timestamp: baseTime})
	if !first.IsNewPattern {
		t.Fatalf("expected first event to create a new pattern")
	}

	for i := 1; i < 4; i++ {
		obs := engine.Process(model.Event{Source: "unit", Raw: "cache miss for user 10", Timestamp: baseTime.Add(time.Duration(i) * 10 * time.Second)})
		if i == 3 && !obs.IsAnomaly {
			t.Fatalf("expected anomaly on repeated spike, got %#v", obs)
		}
	}
}
