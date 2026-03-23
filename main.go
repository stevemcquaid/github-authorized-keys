package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("config", "", "path to config file (default: ~/.config/github-authorized-keys/config.yaml)")
	once := flag.Bool("once", false, "sync once and exit instead of running as a service")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	fetcher := NewFetcher()
	syncer := NewSyncer(cfg.ResolvedKeysPath())

	if *once {
		if err := runSync(cfg, fetcher, syncer); err != nil {
			slog.Error("sync failed", "error", err)
			os.Exit(1)
		}
		return
	}

	// Service loop: sync immediately, then on ticker.
	slog.Info("starting github-authorized-keys",
		"usernames", cfg.GitHubUsernames,
		"interval", cfg.SyncInterval,
		"keys_path", cfg.ResolvedKeysPath(),
	)

	// Initial sync.
	if err := runSync(cfg, fetcher, syncer); err != nil {
		slog.Error("initial sync failed", "error", err)
	}

	ticker := time.NewTicker(cfg.Interval())
	defer ticker.Stop()

	// SIGHUP triggers immediate re-sync (and reloads config).
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)

	// SIGTERM / SIGINT for graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case <-ticker.C:
			if err := runSync(cfg, fetcher, syncer); err != nil {
				slog.Error("sync failed", "error", err)
			}

		case <-sighup:
			slog.Info("received SIGHUP, reloading config and syncing")
			newCfg, err := LoadConfig(*configPath)
			if err != nil {
				slog.Error("failed to reload config", "error", err)
			} else {
				cfg = newCfg
				syncer = NewSyncer(cfg.ResolvedKeysPath())
				ticker.Reset(cfg.Interval())
			}
			if err := runSync(cfg, fetcher, syncer); err != nil {
				slog.Error("sync after SIGHUP failed", "error", err)
			}

		case <-quit:
			slog.Info("shutting down")
			return
		}
	}
}

// runSync fetches keys and writes them to authorized_keys.
func runSync(cfg *Config, fetcher *Fetcher, syncer *Syncer) error {
	slog.Info("syncing keys", "usernames", cfg.GitHubUsernames)

	keys, err := fetcher.FetchKeys(cfg.GitHubUsernames)
	if err != nil {
		return err
	}

	slog.Debug("fetched keys", "count", len(keys))

	if err := syncer.Sync(cfg.GitHubUsernames, keys); err != nil {
		return err
	}

	slog.Info("sync complete", "keys_written", len(keys), "path", syncer.keysPath)
	return nil
}

// newLogger returns a slog.Logger configured for the given level.
// Output goes to stdout so journald captures it naturally.
func newLogger(level string) *slog.Logger {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}
