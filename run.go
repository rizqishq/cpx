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

func findCPPCompiler() (string, string, error) {
	if preferred := strings.TrimSpace(os.Getenv("CXX")); preferred != "" {
		compiler, err := exec.LookPath(preferred)
		if err == nil {
			return filepath.Base(preferred), compiler, nil
		}
		return "", "", fmt.Errorf("preferred compiler from CXX was not found: %s", preferred)
	}

	candidates := []string{"g++", "clang++", "c++"}
	for _, candidate := range candidates {
		compiler, err := exec.LookPath(candidate)
		if err == nil {
			return candidate, compiler, nil
		}
	}
	return "", "", fmt.Errorf("no supported C++ compiler found in PATH; install one of: %s", strings.Join(candidates, ", "))
}

func quoteShellArg(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func detectMSYS2Compiler(compilerPath string) (string, string, bool) {
	dir := filepath.Dir(compilerPath)
	if !strings.EqualFold(filepath.Base(dir), "bin") {
		return "", "", false
	}

	subsystemDir := filepath.Base(filepath.Dir(dir))
	msystemByDir := map[string]string{
		"clang64":    "CLANG64",
		"clangarm64": "CLANGARM64",
		"mingw32":    "MINGW32",
		"mingw64":    "MINGW64",
		"ucrt64":     "UCRT64",
	}

	msystem, ok := msystemByDir[strings.ToLower(subsystemDir)]
	if !ok {
		return "", "", false
	}

	root := filepath.Dir(filepath.Dir(dir))
	return root, msystem, true
}

func prependEnvPath(env []string, dir string) []string {
	if dir == "" {
		return env
	}

	pathKey := "PATH"
	for index, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], pathKey) {
			continue
		}
		current := parts[1]
		env[index] = pathKey + "=" + dir + string(os.PathListSeparator) + current
		return env
	}
	return append(env, pathKey+"="+dir)
}

func compileCommand(sourcePath, binaryPath, compilerPath, standard string) (*exec.Cmd, []string) {
	args := []string{"-std=" + standard, "-O2", "-o", binaryPath, sourcePath}
	cmd := exec.Command(compilerPath, args...)
	if runtime.GOOS == "windows" {
		sourceDir := filepath.Dir(sourcePath)
		if sourceDir == filepath.Dir(binaryPath) {
			args = []string{"-std=" + standard, "-O2", "-o", filepath.Base(binaryPath), filepath.Base(sourcePath)}
			cmd = exec.Command(compilerPath, args...)
			cmd.Dir = sourceDir
		}
	}
	return cmd, args
}

func compileCommandViaMSYS2(sourcePath, binaryPath, compilerPath, standard string) (*exec.Cmd, []string, error) {
	cmd, args := compileCommand(sourcePath, binaryPath, compilerPath, standard)
	if runtime.GOOS != "windows" {
		return cmd, args, nil
	}

	root, msystem, ok := detectMSYS2Compiler(compilerPath)
	if !ok {
		return nil, nil, errors.New("compiler is not from an MSYS2 MinGW environment")
	}

	bashPath := filepath.Join(root, "usr", "bin", "bash.exe")
	commandParts := []string{quoteShellArg(filepath.Base(compilerPath))}
	for _, arg := range args {
		commandParts = append(commandParts, quoteShellArg(arg))
	}

	fallback := exec.Command(bashPath, "-lc", strings.Join(commandParts, " "))
	fallback.Dir = cmd.Dir
	fallback.Env = append(os.Environ(),
		"CHERE_INVOKING=1",
		"MSYSTEM="+msystem,
		"MSYS2_PATH_TYPE=inherit",
	)
	return fallback, args, nil
}

func runtimeEnvForCompiler(compilerPath string) []string {
	env := append([]string{}, os.Environ()...)
	if runtime.GOOS != "windows" {
		return env
	}

	if _, _, ok := detectMSYS2Compiler(compilerPath); ok {
		return prependEnvPath(env, filepath.Dir(compilerPath))
	}
	return env
}

func compileCPP(sourcePath, binaryPath, compilerName, compilerPath, standard string) error {
	cmd, args := compileCommand(sourcePath, binaryPath, compilerPath, standard)
	if runtime.GOOS == "windows" {
		msys2Cmd, msys2Args, msys2Err := compileCommandViaMSYS2(sourcePath, binaryPath, compilerPath, standard)
		if msys2Err == nil {
			cmd = msys2Cmd
			args = msys2Args
		}
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
			message += fmt.Sprintf("\nhint: run `%s --version` and then try the same compile command manually", compilerName)
			if runtime.GOOS == "windows" {
				message += "\nhint: on Windows, this often means the compiler needs a different shell or missing runtime DLLs in PATH"
				if cmd.Dir != "" {
					message += fmt.Sprintf("\nhint: cpx compiled from %s using `%s %s`", cmd.Dir, compilerName, strings.Join(args, " "))
				}
				if strings.EqualFold(filepath.Base(cmd.Path), "bash.exe") {
					message += "\nhint: MSYS2 shell invocation failed too"
				}
			}
		}
		if strings.Contains(message, "bits/stdc++.h") {
			message += "\nhint: replace #include <bits/stdc++.h> with standard headers like #include <iostream>"
		}
		return fmt.Errorf("%s compilation failed (%s): %s", compilerName, compilerPath, message)
	}
	return nil
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

func cmdRun(root, problem string, stdout io.Writer) error {
	cfg, err := loadConfig(root)
	if err != nil {
		return err
	}

	sourceName, err := sourceFileName(cfg)
	if err != nil {
		return err
	}

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
		if writeErr := writeRunFailureHeader(stdout, "Error: compilation failed"); writeErr != nil {
			return writeErr
		}
		if writeErr := writeRunFailureDetail(stdout, "Compiler", fmt.Sprintf("%s (%s)", compilerName, compilerPath)); writeErr != nil {
			return writeErr
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
	if _, err := fmt.Fprintf(stdout, "Compiled %s\n", sourcePath); err != nil {
		return err
	}

	runtimeEnv := runtimeEnvForCompiler(compilerPath)
	passedCount := 0
	for index, pair := range pairs {
		actual, err := runSample(binaryPath, pair[0], runtimeEnv)
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
			if _, writeErr := fmt.Fprintf(stdout, "Stopped after first failed sample.\n"); writeErr != nil {
				return writeErr
			}
			if _, writeErr := fmt.Fprintf(stdout, "Summary: %d/%d passed before stopping at sample %d\n", passedCount, len(pairs), index+1); writeErr != nil {
				return writeErr
			}
			return errRunHandled
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
			if writeErr := writeRunFailureDetail(stdout, "Input file", pair[0]); writeErr != nil {
				return writeErr
			}
			if writeErr := writeRunFailureDetail(stdout, "Expected file", pair[1]); writeErr != nil {
				return writeErr
			}
			if writeErr := writeRunFailureBlock(stdout, "Expected", expectedNormalized); writeErr != nil {
				return writeErr
			}
			if writeErr := writeRunFailureBlock(stdout, "Actual", actualNormalized); writeErr != nil {
				return writeErr
			}
			if _, err := fmt.Fprintf(stdout, "Stopped after first failed sample.\n"); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(stdout, "Summary: %d/%d passed before stopping at sample %d\n", passedCount, len(pairs), index+1); err != nil {
				return err
			}
			return errRunHandled
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
