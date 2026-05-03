package ingest

import (
	"bufio"
	"context"
	"os"
	"strings"
	"time"

	"github.com/hamdyelbatal122/lynxeye/internal/model"
)

type StdinProvider struct {
	options sourceOptions
}

func (s *StdinProvider) Name() string {
	return s.options.name
}

func (s *StdinProvider) Start(ctx context.Context, out chan<- model.Event) error {
	scanner := bufio.NewScanner(os.Stdin)
	buffer := make([]byte, 0, 64*1024)
	scanner.Buffer(buffer, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || shouldIgnore(s.options.ignore, line) {
			continue
		}
		event := model.Event{Source: s.options.name, Raw: line, Timestamp: time.Now().UTC()}
		if err := emitEvent(ctx, out, event); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
