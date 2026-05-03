package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hamdyelbatal122/lynxeye/internal/alert"
	"github.com/hamdyelbatal122/lynxeye/internal/config"
	"github.com/hamdyelbatal122/lynxeye/internal/detect"
	"github.com/hamdyelbatal122/lynxeye/internal/ingest"
	"github.com/hamdyelbatal122/lynxeye/internal/model"
	"github.com/hamdyelbatal122/lynxeye/internal/store"
	"github.com/hamdyelbatal122/lynxeye/internal/ui"
)

func Run(ctx context.Context, configPath string, once bool) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	providers, err := ingest.NewProviders(cfg.Sources, once)
	if err != nil {
		return err
	}

	stateStore, err := store.Open(cfg.State.Path)
	if err != nil {
		return err
	}
	defer stateStore.Close()

	persistedClusters, err := stateStore.LoadClusters()
	if err != nil {
		return err
	}

	engine := detect.NewEngine(cfg.Detection, persistedClusters)
	dispatcher := buildDispatcher(cfg)
	ui.PrintStartup(cfg, len(providers), notifierCount(cfg), len(persistedClusters), once)

	events := make(chan model.Event, cfg.App.BufferSize)
	providerErrors := make(chan error, len(providers))
	providerDone := make(chan struct{})

	var waitGroup sync.WaitGroup
	for _, provider := range providers {
		waitGroup.Add(1)
		go func(provider ingest.Provider) {
			defer waitGroup.Done()
			if err := provider.Start(ctx, events); err != nil && !errors.Is(err, context.Canceled) {
				providerErrors <- fmt.Errorf("provider %s: %w", provider.Name(), err)
			}
		}(provider)
	}

	go func() {
		waitGroup.Wait()
		close(events)
		close(providerDone)
	}()

	for {
		select {
		case <-ctx.Done():
			ui.PrintShutdown()
			return nil
		case err := <-providerErrors:
			return err
		case event, ok := <-events:
			if !ok {
				if once {
					ui.PrintShutdown()
					return nil
				}
				return errors.New("all providers exited unexpectedly")
			}
			observation := engine.Process(event)
			if err := stateStore.SaveCluster(observation.Cluster); err != nil {
				return err
			}
			if observation.IsAnomaly || observation.IsNewPattern {
				ui.PrintObservation(observation)
			}
			if dispatcher.Enabled() && (observation.IsAnomaly || observation.IsNewPattern) {
				alertMessage := toAlert(observation)
				go func() {
					alertCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
					defer cancel()
					if err := dispatcher.Notify(alertCtx, alertMessage); err != nil {
						ui.PrintAlertError(err)
					}
				}()
			}
		}
	}
}

func buildDispatcher(cfg config.Config) *alert.Dispatcher {
	notifiers := make([]alert.Notifier, 0, 2)
	if cfg.Alerts.Slack.Enabled {
		notifiers = append(notifiers, alert.NewSlackNotifier(cfg.Alerts.Slack.WebhookURL))
	}
	if cfg.Alerts.Telegram.Enabled {
		notifiers = append(notifiers, alert.NewTelegramNotifier(cfg.Alerts.Telegram.BotToken, cfg.Alerts.Telegram.ChatID))
	}
	return alert.NewDispatcher(cfg.Alerts.RateLimit, notifiers...)
}

func notifierCount(cfg config.Config) int {
	count := 0
	if cfg.Alerts.Slack.Enabled {
		count++
	}
	if cfg.Alerts.Telegram.Enabled {
		count++
	}
	return count
}

func toAlert(observation model.PatternObservation) model.Alert {
	severity := "warning"
	title := "New log pattern detected"
	if observation.IsAnomaly {
		severity = "critical"
		title = "Log pattern anomaly detected"
	}

	body := fmt.Sprintf(
		"source=%s\nreason=%s\npattern=%s\nsample=%s\nwindow_count=%d\nbaseline=%.2f",
		observation.Event.Source,
		observation.Reason,
		observation.Cluster.Pattern,
		observation.Cluster.Sample,
		observation.WindowCount,
		observation.Baseline,
	)

	return model.Alert{
		Key:      fmt.Sprintf("%s:%s:%s", observation.Event.Source, severity, observation.Cluster.Pattern),
		Title:    title,
		Body:     body,
		Severity: severity,
		Source:   observation.Event.Source,
		Pattern:  observation.Cluster.Pattern,
	}
}
