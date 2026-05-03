package ingest

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hamdyelbatal122/lynxeye/internal/config"
	"github.com/hamdyelbatal122/lynxeye/internal/model"
)

type Provider interface {
	Name() string
	Start(ctx context.Context, out chan<- model.Event) error
}

type sourceOptions struct {
	name          string
	path          string
	readFromStart bool
	pollInterval  time.Duration
	once          bool
	ignore        []*regexp.Regexp
	ssh           config.SSHConfig
}

func NewProviders(sourceConfigs []config.SourceConfig, once bool) ([]Provider, error) {
	providers := make([]Provider, 0, len(sourceConfigs))
	for _, sourceConfig := range sourceConfigs {
		ignorePatterns, err := compilePatterns(sourceConfig.IgnorePatterns)
		if err != nil {
			return nil, fmt.Errorf("compile ignore patterns for source %q: %w", sourceConfig.Name, err)
		}

		options := sourceOptions{
			name:          sourceConfig.Name,
			path:          sourceConfig.Path,
			readFromStart: sourceConfig.ReadFromStart,
			pollInterval:  sourceConfig.PollInterval,
			once:          once,
			ignore:        ignorePatterns,
			ssh:           sourceConfig.SSH,
		}

		switch strings.ToLower(sourceConfig.Type) {
		case "file":
			providers = append(providers, &FileProvider{options: options})
		case "stdin":
			providers = append(providers, &StdinProvider{options: options})
		case "ssh":
			providers = append(providers, &SSHProvider{options: options})
		default:
			return nil, fmt.Errorf("unsupported provider type %q", sourceConfig.Type)
		}
	}
	return providers, nil
}

func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, re)
	}
	return compiled, nil
}

func shouldIgnore(ignore []*regexp.Regexp, line string) bool {
	for _, pattern := range ignore {
		if pattern.MatchString(line) {
			return true
		}
	}
	return false
}

func emitEvent(ctx context.Context, out chan<- model.Event, event model.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case out <- event:
		return nil
	}
}
