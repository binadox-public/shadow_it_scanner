// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
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

// Load reads configuration from file and environment
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

	return cfg, nil
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
