package main

import (
	"fmt"
	"os"
)

const (
	// DefaultDebug is the default debug mode
	DefaultDebug = false

	// DefaultDataDir is the default data directory for storage
	DefaultDataDir = "./data"

	// DefaultAddr is the default bind address of the server
	DefaultAddr = "0.0.0.0:8000"

	// DefaultBaseURL is the default Base URL for the app used to construct feed URLs
	DefaultBaseURL = "http://0.0.0.0:8000"

	// DefaultFeedsFile is the default feeds configuration filename used by the server
	DefaultFeedsFile = "feeds.yaml"

	// DefaultMaxFeedSize is the default maximum feed size before rotation
	DefaultMaxFeedSize = 1 << 19 // ~512KB
)

func NewConfig() *Config {
	return &Config{
		Addr:  DefaultAddr,
		Debug: DefaultDebug,

		DataDir:     DefaultDataDir,
		BaseURL:     DefaultBaseURL,
		FeedsFile:   DefaultFeedsFile,
		MaxFeedSize: DefaultMaxFeedSize,

		Feeds: make(map[string]*Feed),
	}
}

// Option is a function that takes a config struct and modifies it
type Option func(*Config) error

// WithDebug sets the debug mode lfag
func WithDebug(debug bool) Option {
	return func(cfg *Config) error {
		cfg.Debug = debug
		return nil
	}
}

// WithBind sets the server's listening bind address
func WithBind(addr string) Option {
	return func(cfg *Config) error {
		cfg.Addr = addr
		return nil
	}
}

// WithDataDir sets the data directory to use for storage
func WithDataDir(dataDir string) Option {
	return func(cfg *Config) error {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("error creating data-dir %s: %w", dataDir, err)
		}
		cfg.DataDir = dataDir
		return nil
	}
}

// WithFeedsFile set the feeds configuration file used by the server
func WithFeedsFile(feedsFile string) Option {
	return func(cfg *Config) error {
		cfg.FeedsFile = feedsFile
		return nil
	}
}

// WithBaseURL sets the Base URL used for constructing feed URLs
func WithBaseURL(baseURL string) Option {
	return func(cfg *Config) error {
		cfg.BaseURL = baseURL
		return nil
	}
}
