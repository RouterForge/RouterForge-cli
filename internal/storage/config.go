package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type PipelineProfile struct {
	Teams      []string `json:"teams"`
	MaxAgents  int      `json:"max_agents"`
	AutoReview bool     `json:"auto_review"`
}

type PipelineConfig struct {
	Teams    []string                    `json:"teams"`
	Profile  string                      `json:"profile"`
	Profiles map[string]*PipelineProfile `json:"profiles"`
}

type Config struct {
	Model      string          `json:"model"`
	SmallModel string          `json:"small_model"`
	ProjectDir string          `json:"project_dir"`
	DataDir    string          `json:"data_dir"`
	Pipeline   *PipelineConfig `json:"pipeline"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Model:      "zen/big-pickle",
		SmallModel: "zen/deepseek-v4-flash-free",
		ProjectDir: ".",
		DataDir:    filepath.Join(home, ".routerforge"),
		Pipeline: &PipelineConfig{
			Teams:   []string{"backend", "frontend"},
			Profile: "full",
			Profiles: map[string]*PipelineProfile{
				"quick": {
					Teams:      []string{"frontend"},
					MaxAgents:  2,
					AutoReview: true,
				},
				"full": {
					Teams:      []string{"backend", "frontend", "security", "qa"},
					MaxAgents:  5,
					AutoReview: true,
				},
			},
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}

func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
