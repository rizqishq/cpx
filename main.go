package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	appDir       = ".cpx"
	configPath   = ".cpx/config.json"
	templatePath = ".cpx/templates/main.cpp"
)

const defaultTemplate = `#include <bits/stdc++.h>
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

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		printHelp(stdout)
		return 0
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	switch args[0] {
	case "init":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "Error: init does not accept arguments")
			return 1
		}
		if err := cmdInit(cwd, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "new":
		if len(args) < 2 || len(args) > 3 {
			fmt.Fprintln(stderr, "Error: new requires a problem name and an optional sample count")
			return 1
		}
		sampleCount, err := parseSampleCountArg(args[2:])
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		if err := cmdNew(cwd, args[1], sampleCount, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "s":
		if len(args) < 2 || len(args) > 3 {
			fmt.Fprintln(stderr, "Error: s requires a problem name and an optional sample count")
			return 1
		}
		sampleCount, err := parseSampleCountArg(args[2:])
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		if err := cmdAddSamples(cwd, args[1], sampleCount, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "run":
		if len(args) != 2 {
			fmt.Fprintln(stderr, "Error: run requires a problem name")
			return 1
		}
		if err := cmdRun(cwd, args[1], stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "Error: unknown command")
		return 1
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: cpx [command]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  init          initialize competitive programming workspace")
	fmt.Fprintln(w, "  new <problem> [count] create a new problem folder")
	fmt.Fprintln(w, "  s <problem> [count]   add sample files to a problem")
	fmt.Fprintln(w, "  run <problem>         compile and test a problem")
}

func ensureWorkspace(root string) error {
	templatesDir := filepath.Join(root, appDir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		return err
	}

	configFile := filepath.Join(root, configPath)
	if _, err := os.Stat(configFile); errors.Is(err, os.ErrNotExist) {
		data, err := json.MarshalIndent(config{Language: "cpp", Standard: "c++17"}, "", "  ")
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

func readTemplate(root string) ([]byte, error) {
	templateFile := filepath.Join(root, templatePath)
	data, err := os.ReadFile(templateFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("workspace not initialized; run 'cpx init' first")
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

func cmdNew(root, problem string, sampleCount int, stdout io.Writer) error {
	template, err := readTemplate(root)
	if err != nil {
		return err
	}

	problemDir := filepath.Join(root, problem)
	samplesDir := filepath.Join(problemDir, "samples")
	if err := os.Mkdir(problemDir, 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(samplesDir, 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(problemDir, "main.cpp"), template, 0o644); err != nil {
		return err
	}
	if err := createSampleFiles(samplesDir, 1, sampleCount); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "Created problem at %s\n", problemDir)
	return err
}

func cmdAddSamples(root, problem string, sampleCount int, stdout io.Writer) error {
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

func normalizeOutput(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	lines := strings.Split(value, "\n")
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

func samplePairs(samplesDir string) ([][2]string, error) {
	entries, err := os.ReadDir(samplesDir)
	if err != nil {
		return nil, err
	}

	var inputs []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".in") {
			inputs = append(inputs, filepath.Join(samplesDir, entry.Name()))
		}
	}
	sort.Strings(inputs)

	pairs := make([][2]string, 0, len(inputs))
	for _, inputPath := range inputs {
		outputPath := strings.TrimSuffix(inputPath, ".in") + ".out"
		if _, err := os.Stat(outputPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("missing sample output for %s", filepath.Base(inputPath))
			}
			return nil, err
		}
		pairs = append(pairs, [2]string{inputPath, outputPath})
	}
	return pairs, nil
}

func compileCPP(sourcePath, binaryPath string) error {
	compiler, err := exec.LookPath("g++")
	if err != nil {
		return errors.New("g++ is not available")
	}

	cmd := exec.Command(compiler, "-std=c++17", "-O2", "-o", binaryPath, sourcePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = "compilation failed"
		}
		return errors.New(message)
	}
	return nil
}

func runSample(binaryPath, inputPath string) (string, error) {
	input, err := os.ReadFile(inputPath)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(binaryPath)
	cmd.Stdin = bytes.NewReader(input)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("program exited with error: %v", err)
	}
	return stdout.String(), nil
}

func cmdRun(root, problem string, stdout io.Writer) error {
	problemDir := filepath.Join(root, problem)
	sourcePath := filepath.Join(problemDir, "main.cpp")
	samplesDir := filepath.Join(problemDir, "samples")

	if _, err := os.Stat(sourcePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("missing source file: %s", sourcePath)
		}
		return err
	}
	if _, err := os.Stat(samplesDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("missing samples directory: %s", samplesDir)
		}
		return err
	}

	pairs, err := samplePairs(samplesDir)
	if err != nil {
		return err
	}
	if len(pairs) == 0 {
		return errors.New("no sample inputs found")
	}

	tempDir, err := os.MkdirTemp("", "cpx-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	binaryPath := filepath.Join(tempDir, "main")
	if err := compileCPP(sourcePath, binaryPath); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Compiled %s\n", sourcePath); err != nil {
		return err
	}

	allPassed := true
	for index, pair := range pairs {
		actual, err := runSample(binaryPath, pair[0])
		if err != nil {
			return err
		}
		expectedBytes, err := os.ReadFile(pair[1])
		if err != nil {
			return err
		}

		actualNormalized := normalizeOutput(actual)
		expectedNormalized := normalizeOutput(string(expectedBytes))
		status := "FAIL"
		if actualNormalized == expectedNormalized {
			status = "PASS"
		} else {
			allPassed = false
		}

		if _, err := fmt.Fprintf(stdout, "Sample %d: %s\n", index+1, status); err != nil {
			return err
		}
		if status == "FAIL" {
			if _, err := fmt.Fprintln(stdout, "Expected:"); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(stdout, expectedNormalized); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(stdout, "Actual:"); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(stdout, actualNormalized); err != nil {
				return err
			}
		}
	}

	if !allPassed {
		return errors.New("one or more samples failed")
	}
	return nil
}
