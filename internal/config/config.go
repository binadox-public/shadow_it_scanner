// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the scanner
type Config struct {
	ServerURL   string        `mapstructure:"server_url"`
	APIKey      string        `mapstructure:"api_key"`
	InitialDays int           `mapstructure:"initial_days"`
	Timeout     time.Duration `mapstructure:"timeout"`
	ChunkSizeKB int           `mapstructure:"chunk_size_kb"` // Max compressed chunk size in KB
	Compress    bool          `mapstructure:"compress"`      // Enable gzip compression
	StateFile   string        `mapstructure:"state_file"`
	LogFile     string        `mapstructure:"log_file"`
	Source      string        `mapstructure:"source"`

	// discoveredConfig is true if config was obtained via auto-discovery
	discoveredConfig bool
}

// DefaultConfig returns configuration with default values
func DefaultConfig() *Config {
	return &Config{
		InitialDays: 7,
		Timeout:     30 * time.Second,
		ChunkSizeKB: 1024, // 1MB default
		Compress:    true, // Gzip enabled by default
		Source:      "hist_scanner",
	}
}

// Load reads configuration from file, environment, and optionally auto-discovery.
// Priority (highest to lowest): CLI flags > Env vars > Config file > Auto-discovery
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath != "" {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Environment variable overrides
	viper.SetEnvPrefix("HIST_SCANNER")
	viper.AutomaticEnv()

	// Bind config keys
	viper.SetDefault("initial_days", cfg.InitialDays)
	viper.SetDefault("timeout", cfg.Timeout)
	viper.SetDefault("chunk_size_kb", cfg.ChunkSizeKB)
	viper.SetDefault("compress", cfg.Compress)
	viper.SetDefault("source", cfg.Source)

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Try auto-discovery as fallback if ServerURL or APIKey are missing
	if cfg.ServerURL == "" || cfg.APIKey == "" {
		if discovered := Discover(); discovered != nil {
			if cfg.ServerURL == "" {
				cfg.ServerURL = discovered.ServerURL
			}
			if cfg.APIKey == "" {
				cfg.APIKey = discovered.APIKey
			}
			cfg.discoveredConfig = true
		}
	}

	return cfg, nil
}

// WasDiscovered returns true if configuration was obtained via auto-discovery
func (c *Config) WasDiscovered() bool {
	return c.discoveredConfig
}

// Validate checks that required configuration is present
func (c *Config) Validate() error {
	if c.ServerURL == "" {
		return fmt.Errorf("server_url is required")
	}
	if c.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if c.InitialDays < 0 {
		return fmt.Errorf("initial_days must be >= 0")
	}
	if c.ChunkSizeKB <= 0 {
		return fmt.Errorf("chunk_size_kb must be > 0")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be > 0")
	}
	return nil
}

// ApplyFlags merges CLI flag values into config (non-empty values override)
func (c *Config) ApplyFlags(serverURL, apiKey, stateFile, logFile string, initialDays, chunkSizeKB int, compress bool, compressSet bool, timeout time.Duration) {
	if serverURL != "" {
		c.ServerURL = serverURL
	}
	if apiKey != "" {
		c.APIKey = apiKey
	}
	if stateFile != "" {
		c.StateFile = stateFile
	}
	if logFile != "" {
		c.LogFile = logFile
	}
	if initialDays > 0 {
		c.InitialDays = initialDays
	}
	if chunkSizeKB > 0 {
		c.ChunkSizeKB = chunkSizeKB
	}
	if compressSet {
		c.Compress = compress
	}
	if timeout > 0 {
		c.Timeout = timeout
	}
}

// configFile represents the YAML structure for saving config
type configFile struct {
	ServerURL   string `yaml:"server_url"`
	APIKey      string `yaml:"api_key"`
	InitialDays int    `yaml:"initial_days"`
	Timeout     string `yaml:"timeout"`
	ChunkSizeKB int    `yaml:"chunk_size_kb"`
	Compress    bool   `yaml:"compress"`
	StateFile   string `yaml:"state_file,omitempty"`
	LogFile     string `yaml:"log_file,omitempty"`
	Source      string `yaml:"source"`
}

// SaveToFile writes the configuration to a YAML file
func (c *Config) SaveToFile(path string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	cf := configFile{
		ServerURL:   c.ServerURL,
		APIKey:      c.APIKey,
		InitialDays: c.InitialDays,
		Timeout:     c.Timeout.String(),
		ChunkSizeKB: c.ChunkSizeKB,
		Compress:    c.Compress,
		StateFile:   c.StateFile,
		LogFile:     c.LogFile,
		Source:      c.Source,
	}

	data, err := yaml.Marshal(cf)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with restricted permissions (contains API key)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
