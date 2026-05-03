package alert

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hamdyelbatal122/lynxeye/internal/model"
)

type Notifier interface {
	Name() string
	Send(ctx context.Context, alert model.Alert) error
}

type Dispatcher struct {
	notifiers []Notifier
	limiter   *RateLimiter
}

func NewDispatcher(rateLimit time.Duration, notifiers ...Notifier) *Dispatcher {
	return &Dispatcher{
		notifiers: notifiers,
		limiter:   NewRateLimiter(rateLimit),
	}
}

func (d *Dispatcher) Enabled() bool {
	return len(d.notifiers) > 0
}

func (d *Dispatcher) Notify(ctx context.Context, alert model.Alert) error {
	if len(d.notifiers) == 0 {
		return nil
	}

	var errs []string
	for _, notifier := range d.notifiers {
		key := notifier.Name() + ":" + alert.Key
		if !d.limiter.Allow(key) {
			continue
		}
		if err := notifier.Send(ctx, alert); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", notifier.Name(), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("alert delivery failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

type RateLimiter struct {
	mu          sync.Mutex
	lastSeen    map[string]time.Time
	minInterval time.Duration
}

func NewRateLimiter(interval time.Duration) *RateLimiter {
	if interval <= 0 {
		interval = time.Minute
	}
	return &RateLimiter{
		lastSeen:    make(map[string]time.Time),
		minInterval: interval,
	}
}

func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if last, exists := r.lastSeen[key]; exists && now.Sub(last) < r.minInterval {
		return false
	}
	r.lastSeen[key] = now
	return true
}
