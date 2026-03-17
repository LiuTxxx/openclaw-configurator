package config

import (
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/teecert/openclaw-configurator/internal/connection"
)

func ReadConfig(cfs connection.FileSystem, path string) (*OpenClawConfig, error) {
	data, err := cfs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg OpenClawConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.Models.Providers == nil {
		cfg.Models.Providers = make(map[string]*Provider)
	}
	if cfg.Agents.Defaults.Models == nil {
		cfg.Agents.Defaults.Models = make(map[string]json.RawMessage)
	}

	return &cfg, nil
}

func WriteConfig(cfs connection.FileSystem, path string, cfg *OpenClawConfig) error {
	if err := backupConfig(cfs, path); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	data = append(data, '\n')

	if err := cfs.WriteFile(path, data, fs.FileMode(0600)); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

func WriteRawConfig(cfs connection.FileSystem, path string, data []byte) error {
	if err := backupConfig(cfs, path); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}
	return cfs.WriteFile(path, data, fs.FileMode(0600))
}

func backupConfig(cfs connection.FileSystem, path string) error {
	data, err := cfs.ReadFile(path)
	if err != nil {
		return nil
	}
	bakPath := path + ".bak"
	return cfs.WriteFile(bakPath, data, fs.FileMode(0600))
}
