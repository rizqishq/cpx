package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	appDir     = ".cpx"
	configPath = ".cpx/config.json"
)

const legacyDefaultTemplate = `#include <bits/stdc++.h>
using namespace std;

int main() {
    ios::sync_with_stdio(false);
    cin.tie(nullptr);

    return 0;
}
`

const defaultTemplate = `#include <iostream>
using namespace std;

int main() {
    ios::sync_with_stdio(false);
    cin.tie(nullptr);

    return 0;
}
`

type config struct {
	Language string `json:"language"`
	Standard string `json:"standard"`
}

func defaultConfig() config {
	return config{
		Language: "cpp",
		Standard: "c++17",
	}
}

func normalizeConfig(cfg config) config {
	cfg.Language = strings.ToLower(strings.TrimSpace(cfg.Language))
	cfg.Standard = strings.TrimSpace(cfg.Standard)

	defaults := defaultConfig()
	if cfg.Language == "" {
		cfg.Language = defaults.Language
	}
	if cfg.Standard == "" {
		cfg.Standard = defaults.Standard
	}

	return cfg
}

func validateConfig(cfg config) error {
	if _, err := sourceFileName(cfg); err != nil {
		return err
	}
	if cfg.Standard == "" {
		return errors.New("config standard must not be empty")
	}
	return nil
}

func writeConfig(root string, cfg config) error {
	cfg = normalizeConfig(cfg)
	if err := validateConfig(cfg); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(root, configPath), data, 0o644)
}

func loadConfig(root string) (config, error) {
	data, err := os.ReadFile(filepath.Join(root, configPath))
	if errors.Is(err, os.ErrNotExist) {
		return config{}, errors.New("workspace not initialized; run 'cpx init' first")
	}
	if err != nil {
		return config{}, err
	}

	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config{}, fmt.Errorf("invalid config file: %w", err)
	}

	cfg = normalizeConfig(cfg)
	if err := validateConfig(cfg); err != nil {
		return config{}, err
	}

	return cfg, nil
}

func sourceFileName(cfg config) (string, error) {
	switch normalizeConfig(cfg).Language {
	case "cpp":
		return "main.cpp", nil
	default:
		return "", fmt.Errorf("unsupported language in config: %s", cfg.Language)
	}
}

func templateRelativePath(cfg config) (string, error) {
	name, err := sourceFileName(cfg)
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "templates", name), nil
}

func defaultTemplateFor(cfg config) ([]byte, error) {
	switch normalizeConfig(cfg).Language {
	case "cpp":
		return []byte(defaultTemplate), nil
	default:
		return nil, fmt.Errorf("unsupported language in config: %s", cfg.Language)
	}
}

func ensureWorkspace(root string) error {
	if err := os.MkdirAll(filepath.Join(root, appDir, "templates"), 0o755); err != nil {
		return err
	}

	configFile := filepath.Join(root, configPath)
	if _, err := os.Stat(configFile); errors.Is(err, os.ErrNotExist) {
		if err := writeConfig(root, defaultConfig()); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	cfg, err := loadConfig(root)
	if err != nil {
		return err
	}

	templateRelPath, err := templateRelativePath(cfg)
	if err != nil {
		return err
	}
	templateFile := filepath.Join(root, templateRelPath)
	templateData, err := defaultTemplateFor(cfg)
	if err != nil {
		return err
	}

	if _, err := os.Stat(templateFile); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(templateFile, templateData, 0o644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if cfg.Language == "cpp" {
		template, err := os.ReadFile(templateFile)
		if err != nil {
			return err
		}
		if string(template) == legacyDefaultTemplate {
			if err := os.WriteFile(templateFile, templateData, 0o644); err != nil {
				return err
			}
		}
	}

	return nil
}

func readTemplate(root string, cfg config) ([]byte, error) {
	templateRelPath, err := templateRelativePath(cfg)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(root, templateRelPath))
	if errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("workspace template missing; run 'cpx init' first")
	}
	return data, err
}
