package config

import (
	"io"
	"os"

	"github.com/pelletier/go-toml/v2"
)

func Load(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config

	b, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	err = toml.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
