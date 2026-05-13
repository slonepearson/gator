package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"
const filePerm = 0644 // The owner can read and write; everyone else can only read.

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func Read() (Config, error) {
	destination, err := getConfigFilePath()

	if err != nil {
		return Config{}, nil
	}

	data, err := os.ReadFile(destination)
	if err != nil {
		return Config{}, fmt.Errorf("error reading file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("error parsing json: %v", err)
	}

	return config, nil
}

func (c Config) SetUser(user string) error {
	c.CurrentUserName = user
	if err := write(c); err != nil {
		return err
	}
	return nil
}

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(home, configFileName)

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("invalid file path")
	}

	return path, nil
}

func write(cfg Config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error parsing json: %v", err)
	}
	destination, err := getConfigFilePath()

	if err != nil {
		return err
	}
	if err := os.WriteFile(destination, data, filePerm); err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	return nil
}
