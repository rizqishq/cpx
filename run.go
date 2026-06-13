package main

import (
	"bytes"
	"context"
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
	"time"
)

var errRunHandled = errors.New("run failure already printed")

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

func formatRunPath(label, value string) string {
	return fmt.Sprintf("  %s: %s", label, value)
}

func writeRunFailureHeader(stdout io.Writer, title string) error {
	title = strings.Replace(title, "Error:", colorizeErrorLabel(), 1)
	_, err := fmt.Fprintf(stdout, "%s\n", title)
	return err
}

func writeRunFailureDetail(stdout io.Writer, label, value string) error {
	_, err := fmt.Fprintf(stdout, "%s\n", formatRunPath(label, value))
	return err
}

func writeRunFailureBlock(stdout io.Writer, label, value string) error {
	_, err := fmt.Fprintf(stdout, "  %s:\n%s\n", label, formatRunOutput(value))
	return err
}

func writeRunSetupFailure(stdout io.Writer, title string, details [][2]string) error {
	if err := writeRunFailureHeader(stdout, title); err != nil {
		return err
	}
	for _, detail := range details {
		if err := writeRunFailureDetail(stdout, detail[0], detail[1]); err != nil {
			return err
		}
	}
	return nil
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

func runSample(commandPath string, commandArgs []string, workingDir, inputPath string, env []string, timeout time.Duration) (string, error) {
	input, err := os.ReadFile(inputPath)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, commandPath, commandArgs...)
	cmd.Env = env
	cmd.Dir = workingDir
	cmd.Stdin = bytes.NewReader(input)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			message := fmt.Sprintf("program timed out after %s", timeout)
			if stderr.Len() > 0 {
				message += fmt.Sprintf("\nstderr:\n%s", strings.TrimRight(stderr.String(), "\n"))
			}
			return "", errors.New(message)
		}

		message := fmt.Sprintf("program exited with error: %v", err)
		if stderr.Len() > 0 {
			message += fmt.Sprintf("\nstderr:\n%s", strings.TrimRight(stderr.String(), "\n"))
		}
		return "", errors.New(message)
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

func splitOutputLines(value string) []string {
	if value == "" {
		return []string{}
	}
	return strings.Split(value, "\n")
}

func outputLineAt(lines []string, index int) string {
	if index < 0 || index >= len(lines) {
		return "<missing>"
	}
	return lines[index]
}

func mismatchLineInfo(expected, actual string) (int, string, string) {
	expectedLines := splitOutputLines(expected)
	actualLines := splitOutputLines(actual)
	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	for index := 0; index < maxLines; index++ {
		expectedLine := outputLineAt(expectedLines, index)
		actualLine := outputLineAt(actualLines, index)
		if expectedLine != actualLine {
			return index + 1, expectedLine, actualLine
		}
	}
	return 0, "", ""
}

func mismatchPreview(expected, actual string, contextLines int) string {
	expectedLines := splitOutputLines(expected)
	actualLines := splitOutputLines(actual)
	lineNumber, _, _ := mismatchLineInfo(expected, actual)
	if lineNumber == 0 {
		return "<no mismatch preview available>"
	}

	lineIndex := lineNumber - 1
	start := lineIndex - contextLines
	if start < 0 {
		start = 0
	}
	end := lineIndex + contextLines
	maxIndex := len(expectedLines) - 1
	if len(actualLines)-1 > maxIndex {
		maxIndex = len(actualLines) - 1
	}
	if end > maxIndex {
		end = maxIndex
	}

	preview := make([]string, 0, (end-start+1)*2)
	for index := start; index <= end; index++ {
		expectedLine := outputLineAt(expectedLines, index)
		actualLine := outputLineAt(actualLines, index)
		lineLabel := fmt.Sprintf("line %d", index+1)
		if expectedLine == actualLine {
			preview = append(preview, fmt.Sprintf("%s: %s", lineLabel, expectedLine))
			continue
		}
		preview = append(preview, fmt.Sprintf("%s expected: %s", lineLabel, colorizeDiffExpected(expectedLine)))
		preview = append(preview, fmt.Sprintf("%s actual:   %s", lineLabel, colorizeDiffActual(actualLine)))
	}
	return strings.Join(preview, "\n")
}

func cmdRun(root, problem string, stdout io.Writer) error {
	if err := validateProblemPath(problem); err != nil {
		return err
	}

	cfg, err := loadConfig(root)
	if err != nil {
		return err
	}

	spec, err := languageSpecForConfig(cfg)
	if err != nil {
		return err
	}

	sourceName := spec.sourceFileName

	problemDir := filepath.Join(root, problem)
	sourcePath := filepath.Join(problemDir, sourceName)
	samplesDir := filepath.Join(problemDir, "samples")

	if _, err := os.Stat(sourcePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if writeErr := writeRunSetupFailure(stdout, "Error: missing source file", [][2]string{{"Path", sourcePath}}); writeErr != nil {
				return writeErr
			}
			return errRunHandled
		}
		return err
	}
	if _, err := os.Stat(samplesDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if writeErr := writeRunSetupFailure(stdout, "Error: missing samples directory", [][2]string{{"Path", samplesDir}}); writeErr != nil {
				return writeErr
			}
			return errRunHandled
		}
		return err
	}

	pairs, err := samplePairs(samplesDir)
	if err != nil {
		if strings.HasPrefix(err.Error(), "missing sample output for ") {
			missingInput := strings.TrimPrefix(err.Error(), "missing sample output for ")
			inputPath := filepath.Join(samplesDir, missingInput)
			expectedPath := strings.TrimSuffix(inputPath, ".in") + ".out"
			if writeErr := writeRunSetupFailure(stdout, "Error: missing sample output", [][2]string{{"Input file", inputPath}, {"Expected file", expectedPath}}); writeErr != nil {
				return writeErr
			}
			return errRunHandled
		}
		if writeErr := writeRunSetupFailure(stdout, "Error: failed to read sample set", [][2]string{{"Samples directory", samplesDir}}); writeErr != nil {
			return writeErr
		}
		return err
	}
	if len(pairs) == 0 {
		if writeErr := writeRunSetupFailure(stdout, "Error: no sample inputs found", [][2]string{{"Samples directory", samplesDir}}); writeErr != nil {
			return writeErr
		}
		return errRunHandled
	}

	tempDir, err := os.MkdirTemp("", "cpx-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	binaryPath := tempBinaryPath(tempDir, problemDir)
	defer os.Remove(binaryPath)
	runtimeInfo, err := prepareLanguageRuntime(sourcePath, binaryPath, cfg)
	if err != nil {
		title := "Error: compilation failed"
		if spec.id != "cpp" {
			title = "Error: runtime setup failed"
		}
		if writeErr := writeRunFailureHeader(stdout, title); writeErr != nil {
			return writeErr
		}
		if runtimeInfo.toolName != "" || runtimeInfo.toolPath != "" {
			label := runtimeInfo.toolLabel
			if label != "" {
				label = strings.ToUpper(label[:1]) + label[1:]
			}
			if writeErr := writeRunFailureDetail(stdout, label, fmt.Sprintf("%s (%s)", runtimeInfo.toolName, runtimeInfo.toolPath)); writeErr != nil {
				return writeErr
			}
		}
		if writeErr := writeRunFailureDetail(stdout, "Standard", cfg.Standard); writeErr != nil {
			return writeErr
		}
		if writeErr := writeRunFailureDetail(stdout, "Source", sourcePath); writeErr != nil {
			return writeErr
		}
		if writeErr := writeRunFailureBlock(stdout, "Output", err.Error()); writeErr != nil {
			return writeErr
		}
		return errRunHandled
	}
	if _, err := fmt.Fprintf(stdout, "%s %s %s\n", colorizeRunStatus("PASS"), runtimeInfo.setupSummary, sourcePath); err != nil {
		return err
	}

	runtimeEnv := runtimeInfo.env
	runTimeout := time.Duration(cfg.RunTimeoutMs) * time.Millisecond
	passedCount := 0
	for index, pair := range pairs {
		actual, err := runSample(runtimeInfo.commandPath, runtimeInfo.commandArgs, runtimeInfo.workingDir, pair[0], runtimeEnv, runTimeout)
		if err != nil {
			if writeErr := writeRunFailureHeader(stdout, fmt.Sprintf("Sample %d (%s): ERROR", index+1, filepath.Base(pair[0]))); writeErr != nil {
				return writeErr
			}
			if writeErr := writeRunFailureDetail(stdout, "Input file", pair[0]); writeErr != nil {
				return writeErr
			}
			if writeErr := writeRunFailureBlock(stdout, "Runtime error", err.Error()); writeErr != nil {
				return writeErr
			}
			if stopOnFirstFailEnabled(cfg) {
				if _, writeErr := fmt.Fprintf(stdout, "Stopped after first failed sample.\n"); writeErr != nil {
					return writeErr
				}
				if _, writeErr := fmt.Fprintf(stdout, "Summary: %d/%d passed before stopping at sample %d\n", passedCount, len(pairs), index+1); writeErr != nil {
					return writeErr
				}
				return errRunHandled
			}
			continue
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

		if _, err := fmt.Fprintf(stdout, "Sample %d (%s): %s\n", index+1, filepath.Base(pair[0]), colorizeRunStatus(status)); err != nil {
			return err
		}
		if status == "FAIL" {
			if writeErr := writeRunFailureDetail(stdout, "Input file", pair[0]); writeErr != nil {
				return writeErr
			}
			if writeErr := writeRunFailureDetail(stdout, "Expected file", pair[1]); writeErr != nil {
				return writeErr
			}
			lineNumber, expectedLine, actualLine := mismatchLineInfo(expectedNormalized, actualNormalized)
			if lineNumber > 0 {
				if writeErr := writeRunFailureDetail(stdout, "First mismatch", fmt.Sprintf("line %d", lineNumber)); writeErr != nil {
					return writeErr
				}
				if writeErr := writeRunFailureDetail(stdout, "Expected line", expectedLine); writeErr != nil {
					return writeErr
				}
				if writeErr := writeRunFailureDetail(stdout, "Actual line", actualLine); writeErr != nil {
					return writeErr
				}
				if writeErr := writeRunFailureBlock(stdout, "Diff preview", mismatchPreview(expectedNormalized, actualNormalized, cfg.DiffContextLines)); writeErr != nil {
					return writeErr
				}
			}
			if writeErr := writeRunFailureBlock(stdout, "Expected", expectedNormalized); writeErr != nil {
				return writeErr
			}
			if writeErr := writeRunFailureBlock(stdout, "Actual", actualNormalized); writeErr != nil {
				return writeErr
			}
			if stopOnFirstFailEnabled(cfg) {
				if _, err := fmt.Fprintf(stdout, "Stopped after first failed sample.\n"); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(stdout, "Summary: %d/%d passed before stopping at sample %d\n", passedCount, len(pairs), index+1); err != nil {
					return err
				}
				return errRunHandled
			}
		}
	}

	if _, err := fmt.Fprintf(stdout, "Summary: %d/%d passed\n", passedCount, len(pairs)); err != nil {
		return err
	}
	if passedCount != len(pairs) {
		return errRunHandled
	}
	return nil
}
