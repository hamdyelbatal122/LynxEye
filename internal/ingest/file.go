package ingest

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hamdyelbatal122/lynxeye/internal/model"
)

type FileProvider struct {
	options sourceOptions
}

func (f *FileProvider) Name() string {
	return f.options.name
}

func (f *FileProvider) Start(ctx context.Context, out chan<- model.Event) error {
	file, err := os.Open(f.options.path)
	if err != nil {
		return fmt.Errorf("open file source %q: %w", f.options.path, err)
	}
	defer file.Close()

	if !f.options.readFromStart {
		if _, err := file.Seek(0, io.SeekEnd); err != nil {
			return fmt.Errorf("seek file source %q: %w", f.options.path, err)
		}
	}

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err == nil {
			trimmed := strings.TrimRight(line, "\r\n")
			if trimmed != "" && !shouldIgnore(f.options.ignore, trimmed) {
				event := model.Event{Source: f.options.name, Raw: trimmed, Timestamp: time.Now().UTC()}
				if err := emitEvent(ctx, out, event); err != nil {
					return err
				}
			}
			continue
		}

		if !errors.Is(err, io.EOF) {
			return fmt.Errorf("read file source %q: %w", f.options.path, err)
		}

		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !shouldIgnore(f.options.ignore, trimmed) {
			event := model.Event{Source: f.options.name, Raw: trimmed, Timestamp: time.Now().UTC()}
			if err := emitEvent(ctx, out, event); err != nil {
				return err
			}
		}

		if f.options.once {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(f.options.pollInterval):
		}

		stat, statErr := os.Stat(f.options.path)
		if statErr == nil {
			currentOffset, seekErr := file.Seek(0, io.SeekCurrent)
			if seekErr == nil && stat.Size() < currentOffset {
				if err := file.Close(); err != nil {
					return err
				}
				file, err = os.Open(f.options.path)
				if err != nil {
					return fmt.Errorf("reopen file source %q: %w", f.options.path, err)
				}
				reader = bufio.NewReader(file)
			}
		}
	}
}
