package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all runtime configuration for the service.
type Config struct {
	// GitHubUsernames is the list of GitHub usernames whose keys to sync.
	// In YAML: a single string or a list.
	GitHubUsernames []string `yaml:"github_username"`

	// SyncInterval is a Go duration string (e.g. "1h", "30m").
	SyncInterval string `yaml:"sync_interval"`

	// AuthorizedKeysPath overrides the default ~/.ssh/authorized_keys path.
	AuthorizedKeysPath string `yaml:"authorized_keys_path"`

	// LogLevel controls verbosity: debug, info, warn, error.
	LogLevel string `yaml:"log_level"`

	// parsed duration, populated by Validate()
	parsedInterval time.Duration
}

// Interval returns the parsed sync interval duration.
func (c *Config) Interval() time.Duration {
	return c.parsedInterval
}

// ResolvedKeysPath returns the effective authorized_keys path.
func (c *Config) ResolvedKeysPath() string {
	if c.AuthorizedKeysPath != "" {
		return c.AuthorizedKeysPath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/root", ".ssh", "authorized_keys")
	}
	return filepath.Join(home, ".ssh", "authorized_keys")
}

// defaultConfigPaths returns candidate config file locations in priority order.
func defaultConfigPaths() []string {
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		home, _ := os.UserHomeDir()
		xdgConfig = filepath.Join(home, ".config")
	}
	return []string{
		filepath.Join(xdgConfig, "github-authorized-keys", "config.yaml"),
		filepath.Join(xdgConfig, "github-authorized-keys", "config.yml"),
	}
}

// LoadConfig loads configuration from the given path (or auto-detected paths),
// then applies environment variable overrides.
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{
		SyncInterval: "1h",
		LogLevel:     "info",
	}

	// Determine config file to load.
	candidates := defaultConfigPaths()
	if path != "" {
		candidates = []string{path}
	}

	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading config %s: %w", p, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config %s: %w", p, err)
		}
		break
	}

	// Environment variable overrides.
	if v := os.Getenv("GAK_GITHUB_USERNAME"); v != "" {
		cfg.GitHubUsernames = splitUsernames(v)
	}
	if v := os.Getenv("GAK_SYNC_INTERVAL"); v != "" {
		cfg.SyncInterval = v
	}
	if v := os.Getenv("GAK_AUTHORIZED_KEYS_PATH"); v != "" {
		cfg.AuthorizedKeysPath = v
	}
	if v := os.Getenv("GAK_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	return cfg, cfg.Validate()
}

// Validate checks required fields and parses durations.
func (c *Config) Validate() error {
	if len(c.GitHubUsernames) == 0 {
		return fmt.Errorf("github_username is required (set in config file or GAK_GITHUB_USERNAME env var)")
	}
	for _, u := range c.GitHubUsernames {
		if strings.TrimSpace(u) == "" {
			return fmt.Errorf("github_username contains an empty entry")
		}
	}

	d, err := time.ParseDuration(c.SyncInterval)
	if err != nil {
		return fmt.Errorf("invalid sync_interval %q: %w", c.SyncInterval, err)
	}
	if d <= 0 {
		return fmt.Errorf("sync_interval must be positive")
	}
	c.parsedInterval = d
	return nil
}

// UnmarshalYAML allows github_username to be either a string or a list.
func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	// Use an alias to avoid infinite recursion.
	type rawConfig struct {
		GitHubUsername     interface{} `yaml:"github_username"`
		SyncInterval       string      `yaml:"sync_interval"`
		AuthorizedKeysPath string      `yaml:"authorized_keys_path"`
		LogLevel           string      `yaml:"log_level"`
	}

	var raw rawConfig
	if err := value.Decode(&raw); err != nil {
		return err
	}

	c.SyncInterval = raw.SyncInterval
	c.AuthorizedKeysPath = raw.AuthorizedKeysPath
	c.LogLevel = raw.LogLevel

	switch v := raw.GitHubUsername.(type) {
	case string:
		c.GitHubUsernames = splitUsernames(v)
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				c.GitHubUsernames = append(c.GitHubUsernames, s)
			}
		}
	case nil:
		// leave empty; Validate() will catch it
	default:
		return fmt.Errorf("github_username must be a string or list of strings")
	}

	return nil
}

// splitUsernames splits a comma-separated username string into a slice.
func splitUsernames(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
