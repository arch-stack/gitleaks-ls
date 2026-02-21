package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/zricethezav/gitleaks/v8/config"
)

// Config manages gitleaks configuration
type Config struct {
	mu       sync.RWMutex
	path     string
	rootPath string // workspace root path
	cfg      config.Config
	watcher  *fsnotify.Watcher
	onReload func() // Callback when config changes
}

// NewConfig loads config from path or uses defaults
func NewConfig(workspaceRoot string, onReload func()) (*Config, error) {
	c := &Config{
		onReload: onReload,
		rootPath: workspaceRoot,
	}

	// Look for .gitleaks.toml in workspace root
	configPath := filepath.Join(workspaceRoot, ".gitleaks.toml")
	if _, err := os.Stat(configPath); err == nil {
		c.path = configPath
		slog.Info("found gitleaks config", "path", configPath)
	} else {
		slog.Info("no gitleaks config found, using defaults")
	}

	if err := c.load(); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	return c, nil
}

// load reads the configuration
func (c *Config) load() error {
	v := viper.New()
	v.SetConfigType("toml")

	if c.path != "" {
		v.SetConfigFile(c.path)
		if err := v.ReadInConfig(); err != nil {
			slog.Warn("failed to load config, using defaults", "path", c.path, "error", err)
			// Fallback to defaults
			if err := v.ReadConfig(strings.NewReader(config.DefaultConfig)); err != nil {
				return fmt.Errorf("reading default config: %w", err)
			}
		}
	} else {
		if err := v.ReadConfig(strings.NewReader(config.DefaultConfig)); err != nil {
			return fmt.Errorf("reading default config: %w", err)
		}
	}

	var vc config.ViperConfig
	if err := v.Unmarshal(&vc); err != nil {
		return fmt.Errorf("unmarshaling config: %w", err)
	}

	cfg, err := vc.Translate()
	if err != nil {
		return fmt.Errorf("translating config: %w", err)
	}

	if c.path != "" {
		cfg.Path = c.path
	}

	c.mu.Lock()
	c.cfg = cfg
	c.mu.Unlock()
	return nil
}

// GetConfig returns the current gitleaks config
func (c *Config) GetConfig() config.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg
}

// Watch starts watching the config file for changes
func (c *Config) Watch(ctx context.Context) error {
	if c.path == "" {
		return nil // Nothing to watch
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	c.watcher = watcher

	dir := filepath.Dir(c.path)
	if err := watcher.Add(dir); err != nil {
		return fmt.Errorf("watching directory %s: %w", dir, err)
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Name == c.path {
					if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
						slog.Info("config file changed", "path", c.path)
						if err := c.load(); err != nil {
							slog.Error("failed to reload config", "error", err)
						} else {
							if c.onReload != nil {
								c.onReload()
							}
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Error("watcher error", "error", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}
