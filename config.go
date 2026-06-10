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
	Language      string   `json:"language"`
	Standard      string   `json:"standard"`
	Template      string   `json:"template"`
	CompilerFlags []string `json:"compilerFlags"`
}

func defaultConfig() config {
	return config{
		Language:      "cpp",
		Standard:      "c++17",
		Template:      "main",
		CompilerFlags: []string{},
	}
}

func normalizeConfig(cfg config) config {
	cfg.Language = strings.ToLower(strings.TrimSpace(cfg.Language))
	cfg.Standard = strings.TrimSpace(cfg.Standard)
	cfg.Template = strings.TrimSpace(cfg.Template)

	flags := make([]string, 0, len(cfg.CompilerFlags))
	for _, flag := range cfg.CompilerFlags {
		flag = strings.TrimSpace(flag)
		if flag == "" {
			continue
		}
		flags = append(flags, flag)
	}
	cfg.CompilerFlags = flags

	defaults := defaultConfig()
	if cfg.Language == "" {
		cfg.Language = defaults.Language
	}
	if cfg.Standard == "" {
		cfg.Standard = defaults.Standard
	}
	if cfg.Template == "" {
		cfg.Template = defaults.Template
	}

	return cfg
}

func validateTemplateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("template name must not be empty")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid template name: %s", name)
	}
	if strings.ContainsRune(name, os.PathSeparator) || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("invalid template name: %s", name)
	}
	return nil
}

func validateConfig(cfg config) error {
	if _, err := sourceFileName(cfg); err != nil {
		return err
	}
	if cfg.Standard == "" {
		return errors.New("config standard must not be empty")
	}
	if err := validateTemplateName(cfg.Template); err != nil {
		return err
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

func templateFileName(cfg config, templateName string) (string, error) {
	templateName = strings.TrimSpace(templateName)
	if err := validateTemplateName(templateName); err != nil {
		return "", err
	}

	sourceName, err := sourceFileName(cfg)
	if err != nil {
		return "", err
	}
	return templateName + filepath.Ext(sourceName), nil
}

func templateRelativePath(cfg config, templateName string) (string, error) {
	name, err := templateFileName(cfg, templateName)
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

	templateRelPath, err := templateRelativePath(cfg, cfg.Template)
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
	} else if cfg.Language == "cpp" && cfg.Template == defaultConfig().Template {
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

func availableTemplates(root string, cfg config) ([]string, error) {
	sourceName, err := sourceFileName(cfg)
	if err != nil {
		return nil, err
	}

	templatesDir := filepath.Join(root, appDir, "templates")
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(sourceName)
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ext {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ext))
	}
	return names, nil
}

func readTemplate(root string, cfg config, templateName string) ([]byte, error) {
	templateRelPath, err := templateRelativePath(cfg, templateName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(root, templateRelPath))
	if errors.Is(err, os.ErrNotExist) {
		available, listErr := availableTemplates(root, cfg)
		if listErr != nil || len(available) == 0 {
			return nil, fmt.Errorf("template not found: %s", templateName)
		}
		return nil, fmt.Errorf("template not found: %s (available: %s)", templateName, strings.Join(available, ", "))
	}
	return data, err
}
