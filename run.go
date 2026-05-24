package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

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

func formatRunOutput(value string) string {
	if value == "" {
		return "    <empty>"
	}

	lines := strings.Split(value, "\n")
	for index, line := range lines {
		lines[index] = "    " + line
	}
	return strings.Join(lines, "\n")
}

func samplePairs(samplesDir string) ([][2]string, error) {
	entries, err := os.ReadDir(samplesDir)
	if err != nil {
		return nil, err
	}

	type sampleInput struct {
		number int
		path   string
	}

	var inputs []sampleInput
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".in") {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), ".in")
		number, err := strconv.Atoi(base)
		if err != nil {
			continue
		}
		inputs = append(inputs, sampleInput{
			number: number,
			path:   filepath.Join(samplesDir, entry.Name()),
		})
	}
	sort.Slice(inputs, func(i, j int) bool {
		return inputs[i].number < inputs[j].number
	})

	pairs := make([][2]string, 0, len(inputs))
	for _, input := range inputs {
		outputPath := strings.TrimSuffix(input.path, ".in") + ".out"
		if _, err := os.Stat(outputPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("missing sample output for %s", filepath.Base(input.path))
			}
			return nil, err
		}
		pairs = append(pairs, [2]string{input.path, outputPath})
	}
	return pairs, nil
}

func runSample(binaryPath, inputPath string, env []string) (string, error) {
	input, err := os.ReadFile(inputPath)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(binaryPath)
	cmd.Env = env
	cmd.Dir = filepath.Dir(binaryPath)
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

func tempBinaryPath(tempDir, problemDir string) string {
	name := "main"
	dir := tempDir
	if runtime.GOOS == "windows" {
		name = "cpx-run.exe"
		dir = problemDir
	}
	return filepath.Join(dir, name)
}

func cmdRun(root, problem string, stdout io.Writer) error {
	cfg, err := readConfig(root)
	if err != nil {
		return err
	}

	problemDir := filepath.Join(root, problem)
	sourcePath := filepath.Join(problemDir, sourceFileName(cfg))
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

	compilerName, compilerPath, err := findCPPCompiler()
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "cpx-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	binaryPath := tempBinaryPath(tempDir, problemDir)
	defer os.Remove(binaryPath)
	if err := compileCPP(sourcePath, binaryPath, compilerName, compilerPath, cfg.Standard); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Compiled %s\n", sourcePath); err != nil {
		return err
	}

	runtimeEnv := runtimeEnvForCompiler(compilerPath)
	passedCount := 0
	for index, pair := range pairs {
		actual, err := runSample(binaryPath, pair[0], runtimeEnv)
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
			passedCount++
		}

		if _, err := fmt.Fprintf(stdout, "Sample %d (%s): %s\n", index+1, filepath.Base(pair[0]), status); err != nil {
			return err
		}
		if status == "FAIL" {
			if _, err := fmt.Fprintf(stdout, "  Expected:\n%s\n", formatRunOutput(expectedNormalized)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(stdout, "  Actual:\n%s\n", formatRunOutput(actualNormalized)); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintf(stdout, "Summary: %d/%d passed\n", passedCount, len(pairs)); err != nil {
		return err
	}
	if passedCount != len(pairs) {
		return errors.New("one or more samples failed")
	}
	return nil
}
