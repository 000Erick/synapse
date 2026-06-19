package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all environment-driven configuration for Synapse.
type Config struct {
	OpenAIAPIKey     string
	OpenAIEmbedModel string
	// EmbedEndpoint overrides the embeddings API URL. Empty means the default
	// OpenAI endpoint. Point it at any OpenAI-compatible server (Ollama,
	// LocalAI, LM Studio) to embed with a free, local model.
	EmbedEndpoint string
	// EmbedDims overrides the reported embedding dimensionality. 0 means "use
	// the OpenAI default". Set it to the dimension of your local model (e.g.
	// 768 for nomic-embed-text) so stats report the right number.
	EmbedDims     int
	SynapseDBPath string
	EngramDBPath  string
}

// Load reads configuration from environment variables, applying defaults.
func Load() (*Config, error) {
	cfg := &Config{
		OpenAIAPIKey:     os.Getenv("OPENAI_API_KEY"),
		OpenAIEmbedModel: envOrDefault("OPENAI_EMBED_MODEL", "text-embedding-3-large"),
		EmbedEndpoint:    os.Getenv("SYNAPSE_EMBED_ENDPOINT"),
		SynapseDBPath:    envOrDefault("SYNAPSE_DB_PATH", "~/.synapse/synapse.db"),
		EngramDBPath:     envOrDefault("ENGRAM_DB_PATH", "~/.engram/engram.db"),
	}

	dims, err := envIntOrDefault("SYNAPSE_EMBED_DIMS", 0)
	if err != nil {
		return nil, fmt.Errorf("config: SYNAPSE_EMBED_DIMS: %w", err)
	}
	cfg.EmbedDims = dims

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

// envIntOrDefault parses a non-negative integer from the environment. An empty
// value yields def; a malformed or negative value is a hard error so a typo in
// SYNAPSE_EMBED_DIMS fails loudly instead of silently falling back.
func envIntOrDefault(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%q is not a valid integer", v)
	}
	if n < 0 {
		return 0, fmt.Errorf("%q must not be negative", v)
	}
	return n, nil
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
