package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App       AppConfig       `yaml:"app"`
	State     StateConfig     `yaml:"state"`
	Detection DetectionConfig `yaml:"detection"`
	Alerts    AlertsConfig    `yaml:"alerts"`
	Sources   []SourceConfig  `yaml:"sources"`
}

type AppConfig struct {
	Name       string `yaml:"name"`
	BufferSize int    `yaml:"buffer_size"`
}

type StateConfig struct {
	Path string `yaml:"path"`
}

type DetectionConfig struct {
	Window            time.Duration `yaml:"window"`
	MinEvents         int           `yaml:"min_events"`
	SpikeMultiplier   float64       `yaml:"spike_multiplier"`
	EWMAAlpha         float64       `yaml:"ewma_alpha"`
	AlertOnNewPattern bool          `yaml:"alert_on_new_pattern"`
}

type AlertsConfig struct {
	RateLimit time.Duration  `yaml:"rate_limit"`
	Slack     SlackConfig    `yaml:"slack"`
	Telegram  TelegramConfig `yaml:"telegram"`
}

type SlackConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
}

type TelegramConfig struct {
	Enabled  bool   `yaml:"enabled"`
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

type SourceConfig struct {
	Name           string        `yaml:"name"`
	Type           string        `yaml:"type"`
	Path           string        `yaml:"path"`
	ReadFromStart  bool          `yaml:"read_from_start"`
	PollInterval   time.Duration `yaml:"poll_interval"`
	IgnorePatterns []string      `yaml:"ignore_patterns"`
	SSH            SSHConfig     `yaml:"ssh"`
}

type SSHConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	PrivateKeyPath string `yaml:"private_key_path"`
	KnownHostsPath string `yaml:"known_hosts_path"`
	Command        string `yaml:"command"`
}

func Load(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	applyDefaults(&cfg)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.App.Name == "" {
		cfg.App.Name = "LynxEye"
	}
	if cfg.App.BufferSize <= 0 {
		cfg.App.BufferSize = 4096
	}
	if cfg.State.Path == "" {
		cfg.State.Path = "./state/patterns.db"
	}
	if cfg.Detection.Window <= 0 {
		cfg.Detection.Window = 5 * time.Minute
	}
	if cfg.Detection.MinEvents <= 0 {
		cfg.Detection.MinEvents = 8
	}
	if cfg.Detection.SpikeMultiplier <= 0 {
		cfg.Detection.SpikeMultiplier = 3.0
	}
	if cfg.Detection.EWMAAlpha <= 0 || cfg.Detection.EWMAAlpha > 1 {
		cfg.Detection.EWMAAlpha = 0.35
	}
	if cfg.Alerts.RateLimit <= 0 {
		cfg.Alerts.RateLimit = 15 * time.Minute
	}
	for index := range cfg.Sources {
		if cfg.Sources[index].PollInterval <= 0 {
			cfg.Sources[index].PollInterval = time.Second
		}
		if cfg.Sources[index].SSH.Port == 0 {
			cfg.Sources[index].SSH.Port = 22
		}
	}
}

func (c Config) Validate() error {
	if len(c.Sources) == 0 {
		return errors.New("config must define at least one source")
	}
	if c.Detection.SpikeMultiplier < 1 {
		return errors.New("detection.spike_multiplier must be >= 1")
	}
	if c.Detection.EWMAAlpha <= 0 || c.Detection.EWMAAlpha > 1 {
		return errors.New("detection.ewma_alpha must be within (0, 1]")
	}
	if c.Alerts.Slack.Enabled && c.Alerts.Slack.WebhookURL == "" {
		return errors.New("alerts.slack.webhook_url is required when Slack is enabled")
	}
	if c.Alerts.Telegram.Enabled {
		if c.Alerts.Telegram.BotToken == "" || c.Alerts.Telegram.ChatID == "" {
			return errors.New("alerts.telegram.bot_token and alerts.telegram.chat_id are required when Telegram is enabled")
		}
	}
	for _, source := range c.Sources {
		sourceType := strings.ToLower(strings.TrimSpace(source.Type))
		if source.Name == "" {
			return errors.New("each source must have a name")
		}
		switch sourceType {
		case "file":
			if source.Path == "" {
				return fmt.Errorf("file source %q requires path", source.Name)
			}
		case "stdin":
		case "ssh":
			if source.Path == "" {
				return fmt.Errorf("ssh source %q requires path", source.Name)
			}
			if source.SSH.Host == "" || source.SSH.User == "" {
				return fmt.Errorf("ssh source %q requires ssh.host and ssh.user", source.Name)
			}
			if source.SSH.Password == "" && source.SSH.PrivateKeyPath == "" {
				return fmt.Errorf("ssh source %q requires either ssh.password or ssh.private_key_path", source.Name)
			}
		default:
			return fmt.Errorf("source %q has unsupported type %q", source.Name, source.Type)
		}
	}
	return nil
}
