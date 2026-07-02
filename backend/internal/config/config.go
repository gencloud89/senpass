package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config chứa toàn bộ cấu hình ứng dụng
type Config struct {
	DatabasePath string `yaml:"database-path"`
	Listen       string `yaml:"listen"`
	Key          string `yaml:"key"`
}

// Load đọc cấu hình từ file YAML
func Load(path string) (*Config, error) {
	cfg := &Config{
		DatabasePath: "sqlite3://data.db",
		Listen:       "127.0.0.1:18888",
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// Dùng default nếu không có file
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
