package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

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

	const pathKey = "PATH"
	for index, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], pathKey) {
			continue
		}
		env[index] = pathKey + "=" + dir + string(os.PathListSeparator) + parts[1]
		return env
	}
	return append(env, pathKey+"="+dir)
}

func compileArgs(standard, binaryPath, sourcePath string) []string {
	return []string{"-std=" + standard, "-O2", "-o", binaryPath, sourcePath}
}

func compileCommand(sourcePath, binaryPath, compilerPath, standard string) (*exec.Cmd, []string) {
	args := compileArgs(standard, binaryPath, sourcePath)
	cmd := exec.Command(compilerPath, args...)
	if runtime.GOOS == "windows" {
		sourceDir := filepath.Dir(sourcePath)
		if sourceDir == filepath.Dir(binaryPath) {
			args = compileArgs(standard, filepath.Base(binaryPath), filepath.Base(sourcePath))
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

	cmd = exec.Command(bashPath, "-lc", strings.Join(commandParts, " "))
	cmd.Dir = filepath.Dir(sourcePath)
	cmd.Env = append(os.Environ(),
		"CHERE_INVOKING=1",
		"MSYSTEM="+msystem,
		"MSYS2_PATH_TYPE=inherit",
	)
	return cmd, args, nil
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
	if err == nil {
		return nil
	}

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
