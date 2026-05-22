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
	DbUrl              string `json:"db_url"`
	CurrentUserName    string `json:"current_user_name"`
	LastReadTop        string `json:"last_read_top"`
	LastReadTopUuid    string `json:"last_read_top_uuid"`
	LastReadBottom     string `json:"last_read_bottom"`
	LastReadBottomUuid string `json:"last_read_bottom_uuid"`
}

func Read() (Config, error) {
	destination, err := getConfigFilePath()

	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(destination)
	if err != nil {
		return Config{}, fmt.Errorf("problem reading file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("problem parsing json: %w", err)
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

func (c Config) SetLastRead(top string, bottom string, topUuid string, bottomUuid string) error {
	c.LastReadTop = top
	c.LastReadBottom = bottom
	c.LastReadTopUuid = topUuid
	c.LastReadBottomUuid = bottomUuid
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
		return "", fmt.Errorf("path to invalid file")
	}

	return path, nil
}

func write(cfg Config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("problem parsing json: %w", err)
	}
	destination, err := getConfigFilePath()

	if err != nil {
		return err
	}
	if err := os.WriteFile(destination, data, filePerm); err != nil {
		return fmt.Errorf("problem writing to file: %w", err)
	}

	return nil
}
