package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func ensureWorkspace(root string) error {
	templatesDir := filepath.Join(root, appDir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		return err
	}

	configFile := filepath.Join(root, configPath)
	if _, err := os.Stat(configFile); errors.Is(err, os.ErrNotExist) {
		data, err := json.MarshalIndent(defaultWorkspaceConfig, "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if err := os.WriteFile(configFile, data, 0o644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	templateFile := filepath.Join(root, templatePath)
	if _, err := os.Stat(templateFile); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(templateFile, []byte(defaultTemplate), 0o644); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		template, err := os.ReadFile(templateFile)
		if err != nil {
			return err
		}
		if string(template) == legacyDefaultTemplate {
			if err := os.WriteFile(templateFile, []byte(defaultTemplate), 0o644); err != nil {
				return err
			}
		}
	}

	return nil
}

func cmdInit(root string, stdout io.Writer) error {
	if err := ensureWorkspace(root); err != nil {
		return err
	}
	_, err := fmt.Fprintf(stdout, "Initialized workspace at %s\n", filepath.Join(root, appDir))
	return err
}

func normalizeConfig(cfg config) config {
	normalized := defaultWorkspaceConfig

	if value := strings.TrimSpace(cfg.Language); value != "" {
		normalized.Language = strings.ToLower(value)
	}
	switch normalized.Language {
	case "c++", "cxx":
		normalized.Language = "cpp"
	}

	if value := strings.TrimSpace(cfg.Standard); value != "" {
		normalized.Standard = strings.ToLower(value)
	}

	return normalized
}

func validateConfig(cfg config) error {
	if cfg.Language != "cpp" {
		return fmt.Errorf("unsupported language %q in %s; currently only \"cpp\" is supported", cfg.Language, configPath)
	}
	return nil
}

func readConfig(root string) (config, error) {
	configFile := filepath.Join(root, configPath)
	data, err := os.ReadFile(configFile)
	if errors.Is(err, os.ErrNotExist) {
		return config{}, errors.New("workspace not initialized; run 'cpx init' first")
	}
	if err != nil {
		return config{}, err
	}

	cfg := defaultWorkspaceConfig
	if len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return config{}, fmt.Errorf("invalid config file %s: %w", configPath, err)
		}
	}

	cfg = normalizeConfig(cfg)
	if err := validateConfig(cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func normalizeTemplateName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "main.cpp"
	}
	if filepath.Ext(name) == "" {
		name += ".cpp"
	}
	return name
}

func sourceFileName(cfg config) string {
	switch cfg.Language {
	case "cpp":
		return "main.cpp"
	default:
		return "main.cpp"
	}
}

func validateProblemName(problem string) error {
	name := strings.TrimSpace(problem)
	if name == "" {
		return errors.New("problem name must not be empty")
	}
	if name == "." || name == ".." {
		return errors.New("problem name must not be . or ..")
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return errors.New("problem name must be a simple folder name, not a path")
	}
	return nil
}

func parseNewArgs(args []string) (int, string, error) {
	if len(args) == 0 {
		return 1, "", nil
	}
	if len(args) > 2 {
		return 0, "", errors.New("new accepts at most a sample count and a template name")
	}

	templateName := ""
	sampleCount := 1

	if count, err := strconv.Atoi(args[0]); err == nil {
		if count < 1 {
			return 0, "", errors.New("sample count must be a positive integer")
		}
		sampleCount = count
		if len(args) == 2 {
			templateName = args[1]
		}
		return sampleCount, templateName, nil
	}

	if len(args) == 2 {
		return 0, "", errors.New("if sample count is provided, it must come before the template name")
	}

	templateName = args[0]
	return sampleCount, templateName, nil
}

func parseContestArgs(args []string) ([]string, int, string, error) {
	if len(args) == 0 {
		return nil, 0, "", errors.New("contest requires at least one problem name")
	}

	sampleCount := 1
	templateName := ""
	countFlagSeen := false
	templateFlagSeen := false

	seen := make(map[string]struct{}, len(args))
	problems := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "-c", "--count":
			if countFlagSeen {
				return nil, 0, "", errors.New("contest sample count flag may only be provided once")
			}
			if index+1 >= len(args) {
				return nil, 0, "", fmt.Errorf("missing value for %s", arg)
			}
			count, err := strconv.Atoi(args[index+1])
			if err != nil || count < 1 {
				return nil, 0, "", errors.New("sample count must be a positive integer")
			}
			sampleCount = count
			countFlagSeen = true
			index++
			continue
		case "-t", "--template":
			if templateFlagSeen {
				return nil, 0, "", errors.New("contest template flag may only be provided once")
			}
			if index+1 >= len(args) {
				return nil, 0, "", fmt.Errorf("missing value for %s", arg)
			}
			if strings.TrimSpace(args[index+1]) == "" {
				return nil, 0, "", errors.New("template name must not be empty")
			}
			templateName = args[index+1]
			templateFlagSeen = true
			index++
			continue
		}

		if strings.HasPrefix(arg, "-") {
			return nil, 0, "", fmt.Errorf("unknown contest flag %q", arg)
		}

		problem := arg
		if err := validateProblemName(problem); err != nil {
			return nil, 0, "", err
		}
		if _, ok := seen[problem]; ok {
			return nil, 0, "", fmt.Errorf("duplicate problem name %q", problem)
		}
		seen[problem] = struct{}{}
		problems = append(problems, problem)
	}

	if len(problems) == 0 {
		return nil, 0, "", errors.New("contest requires at least one problem name")
	}

	return problems, sampleCount, templateName, nil
}

func readTemplate(root, templateName string) ([]byte, error) {
	templateFile := filepath.Join(root, appDir, "templates", normalizeTemplateName(templateName))
	data, err := os.ReadFile(templateFile)
	if errors.Is(err, os.ErrNotExist) {
		if templateName == "" {
			return nil, errors.New("workspace not initialized; run 'cpx init' first")
		}
		return nil, fmt.Errorf("missing template %q; expected %s", templateName, templateFile)
	}
	return data, err
}

func parseSampleCountArg(args []string) (int, error) {
	if len(args) == 0 {
		return 1, nil
	}

	count, err := strconv.Atoi(args[0])
	if err != nil || count < 1 {
		return 0, errors.New("sample count must be a positive integer")
	}
	return count, nil
}

func createSampleFiles(samplesDir string, start, count int) error {
	for index := 0; index < count; index++ {
		sampleNumber := start + index
		inputPath := filepath.Join(samplesDir, fmt.Sprintf("%d.in", sampleNumber))
		outputPath := filepath.Join(samplesDir, fmt.Sprintf("%d.out", sampleNumber))
		if err := os.WriteFile(inputPath, []byte{}, 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(outputPath, []byte{}, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func nextSampleNumber(samplesDir string) (int, error) {
	entries, err := os.ReadDir(samplesDir)
	if err != nil {
		return 0, err
	}

	maxSample := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".in") {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), ".in")
		number, err := strconv.Atoi(base)
		if err != nil {
			continue
		}
		if number > maxSample {
			maxSample = number
		}
	}
	return maxSample + 1, nil
}

func createProblem(problemDir, sourcePath string, template []byte, sampleCount int) error {
	created := false
	defer func() {
		if !created {
			_ = os.RemoveAll(problemDir)
		}
	}()

	samplesDir := filepath.Join(problemDir, "samples")
	if err := os.Mkdir(problemDir, 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(samplesDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(sourcePath, template, 0o644); err != nil {
		return err
	}
	if err := createSampleFiles(samplesDir, 1, sampleCount); err != nil {
		return err
	}
	created = true
	return nil
}

func cleanupProblems(root string, problems []string) {
	for _, problem := range problems {
		_ = os.RemoveAll(filepath.Join(root, problem))
	}
}

func cmdNew(root, problem string, sampleCount int, templateName string, stdout io.Writer) error {
	if err := validateProblemName(problem); err != nil {
		return err
	}

	cfg, err := readConfig(root)
	if err != nil {
		return err
	}

	template, err := readTemplate(root, templateName)
	if err != nil {
		return err
	}

	problemDir := filepath.Join(root, problem)
	sourcePath := filepath.Join(problemDir, sourceFileName(cfg))
	if err := createProblem(problemDir, sourcePath, template, sampleCount); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "Created problem at %s\n", problemDir)
	if err != nil {
		cleanupProblems(root, []string{problem})
	}
	return err
}

func cmdContest(root string, problems []string, sampleCount int, templateName string, stdout io.Writer) error {
	cfg, err := readConfig(root)
	if err != nil {
		return err
	}

	template, err := readTemplate(root, templateName)
	if err != nil {
		return err
	}

	for _, problem := range problems {
		problemDir := filepath.Join(root, problem)
		if _, err := os.Stat(problemDir); err == nil {
			return fmt.Errorf("problem already exists: %s", problemDir)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	createdProblems := make([]string, 0, len(problems))
	for _, problem := range problems {
		problemDir := filepath.Join(root, problem)
		sourcePath := filepath.Join(problemDir, sourceFileName(cfg))
		if err := createProblem(problemDir, sourcePath, template, sampleCount); err != nil {
			cleanupProblems(root, createdProblems)
			return err
		}
		createdProblems = append(createdProblems, problem)
		if _, err := fmt.Fprintf(stdout, "Created problem at %s\n", problemDir); err != nil {
			cleanupProblems(root, createdProblems)
			return err
		}
	}

	_, err = fmt.Fprintf(stdout, "Created %d contest problem(s)\n", len(problems))
	return err
}

func cmdAddSamples(root, problem string, sampleCount int, stdout io.Writer) error {
	if err := validateProblemName(problem); err != nil {
		return err
	}

	samplesDir := filepath.Join(root, problem, "samples")
	if _, err := os.Stat(samplesDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("missing samples directory: %s", samplesDir)
		}
		return err
	}

	start, err := nextSampleNumber(samplesDir)
	if err != nil {
		return err
	}
	if err := createSampleFiles(samplesDir, start, sampleCount); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "Added %d sample(s) to %s\n", sampleCount, problem)
	return err
}
