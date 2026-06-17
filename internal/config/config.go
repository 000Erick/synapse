package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds all environment-driven configuration for Synapse.
type Config struct {
	OpenAIAPIKey     string
	OpenAIEmbedModel string
	SynapseDBPath    string
	EngramDBPath     string
}

// Load reads configuration from environment variables, applying defaults.
func Load() (*Config, error) {
	cfg := &Config{
		OpenAIAPIKey:     os.Getenv("OPENAI_API_KEY"),
		OpenAIEmbedModel: envOrDefault("OPENAI_EMBED_MODEL", "text-embedding-3-large"),
		SynapseDBPath:    envOrDefault("SYNAPSE_DB_PATH", "~/.synapse/synapse.db"),
		EngramDBPath:     envOrDefault("ENGRAM_DB_PATH", "~/.engram/engram.db"),
	}

	var err error
	cfg.SynapseDBPath, err = expandHome(cfg.SynapseDBPath)
	if err != nil {
		return nil, fmt.Errorf("config: expand SYNAPSE_DB_PATH: %w", err)
	}
	cfg.EngramDBPath, err = expandHome(cfg.EngramDBPath)
	if err != nil {
		return nil, fmt.Errorf("config: expand ENGRAM_DB_PATH: %w", err)
	}

	return cfg, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func expandHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, path[1:]), nil
}
