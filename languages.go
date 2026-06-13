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

type languageSpec struct {
	id              string
	sourceFileName  string
	defaultTemplate string
	toolLabel       string
	resolveTool     func(cfg config) (string, string, error)
	prepareRuntime  func(sourcePath, binaryPath string, cfg config) (languageRuntime, error)
}

type languageRuntime struct {
	toolLabel    string
	toolName     string
	toolPath     string
	commandPath  string
	commandArgs  []string
	workingDir   string
	env          []string
	setupSummary string
}

func languageSpecForConfig(cfg config) (languageSpec, error) {
	switch normalizeConfig(cfg).Language {
	case "cpp":
		return languageSpec{
			id:              "cpp",
			sourceFileName:  "main.cpp",
			defaultTemplate: defaultTemplate,
			toolLabel:       "compiler",
			resolveTool:     resolveCPPCompiler,
			prepareRuntime:  prepareCPPRuntime,
		}, nil
	case "python":
		return languageSpec{
			id:              "python",
			sourceFileName:  "main.py",
			defaultTemplate: pythonDefaultTemplate,
			toolLabel:       "interpreter",
			resolveTool:     resolvePythonInterpreter,
			prepareRuntime:  preparePythonRuntime,
		}, nil
	default:
		return languageSpec{}, fmt.Errorf("unsupported language in config: %s", cfg.Language)
	}
}

func resolveRuntimeToolForConfig(cfg config) (languageSpec, string, string, error) {
	spec, err := languageSpecForConfig(cfg)
	if err != nil {
		return languageSpec{}, "", "", err
	}
	if spec.resolveTool == nil {
		return spec, "", "", nil
	}
	name, path, err := spec.resolveTool(cfg)
	return spec, name, path, err
}

func prepareLanguageRuntime(sourcePath, binaryPath string, cfg config) (languageRuntime, error) {
	spec, err := languageSpecForConfig(cfg)
	if err != nil {
		return languageRuntime{}, err
	}
	if spec.prepareRuntime == nil {
		return languageRuntime{env: append([]string{}, os.Environ()...)}, nil
	}
	return spec.prepareRuntime(sourcePath, binaryPath, cfg)
}

func resolveCPPCompiler(cfg config) (string, string, error) {
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

func prepareCPPRuntime(sourcePath, binaryPath string, cfg config) (languageRuntime, error) {
	compilerName, compilerPath, err := resolveCPPCompiler(cfg)
	runtimeInfo := languageRuntime{
		toolLabel:    "compiler",
		toolName:     compilerName,
		toolPath:     compilerPath,
		commandPath:  binaryPath,
		commandArgs:  nil,
		workingDir:   filepath.Dir(binaryPath),
		env:          runtimeEnvForCompiler(compilerPath),
		setupSummary: "Compiled",
	}
	if err != nil {
		return runtimeInfo, err
	}
	if err := compileCPP(sourcePath, binaryPath, compilerName, compilerPath, cfg); err != nil {
		return runtimeInfo, err
	}
	return runtimeInfo, nil
}

func resolvePythonInterpreter(cfg config) (string, string, error) {
	interpreterPath, err := exec.LookPath("python3")
	if err != nil {
		return "", "", fmt.Errorf("python3 was not found in PATH")
	}
	return "python3", interpreterPath, nil
}

func preparePythonRuntime(sourcePath, binaryPath string, cfg config) (languageRuntime, error) {
	_ = binaryPath
	interpreterName, interpreterPath, err := resolvePythonInterpreter(cfg)
	runtimeInfo := languageRuntime{
		toolLabel:    "interpreter",
		toolName:     interpreterName,
		toolPath:     interpreterPath,
		commandPath:  interpreterPath,
		commandArgs:  []string{sourcePath},
		workingDir:   filepath.Dir(sourcePath),
		env:          append([]string{}, os.Environ()...),
		setupSummary: "Prepared",
	}
	if err != nil {
		return runtimeInfo, err
	}
	return runtimeInfo, nil
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

func compileArgs(sourcePath, binaryPath, standard string, compilerFlags []string) []string {
	args := []string{"-std=" + standard, "-O2"}
	args = append(args, compilerFlags...)
	args = append(args, "-o", binaryPath, sourcePath)
	return args
}

func compileCommand(sourcePath, binaryPath, compilerPath string, cfg config) (*exec.Cmd, []string) {
	args := compileArgs(sourcePath, binaryPath, cfg.Standard, cfg.CompilerFlags)
	cmd := exec.Command(compilerPath, args...)
	if runtime.GOOS == "windows" {
		sourceDir := filepath.Dir(sourcePath)
		if sourceDir == filepath.Dir(binaryPath) {
			args = compileArgs(filepath.Base(sourcePath), filepath.Base(binaryPath), cfg.Standard, cfg.CompilerFlags)
			cmd = exec.Command(compilerPath, args...)
			cmd.Dir = sourceDir
		}
	}
	return cmd, args
}

func compileCommandViaMSYS2(sourcePath, binaryPath, compilerPath string, cfg config) (*exec.Cmd, []string, error) {
	cmd, args := compileCommand(sourcePath, binaryPath, compilerPath, cfg)
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

func compileCPP(sourcePath, binaryPath, compilerName, compilerPath string, cfg config) error {
	cmd, args := compileCommand(sourcePath, binaryPath, compilerPath, cfg)
	if runtime.GOOS == "windows" {
		msys2Cmd, msys2Args, msys2Err := compileCommandViaMSYS2(sourcePath, binaryPath, compilerPath, cfg)
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
