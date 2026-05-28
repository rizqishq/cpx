package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type doctorStats struct {
	failures int
	warnings int
}

func doctorLine(w io.Writer, status, name, detail string) error {
	if detail == "" {
		_, err := fmt.Fprintf(w, "[%s] %s\n", status, name)
		return err
	}
	_, err := fmt.Fprintf(w, "[%s] %s: %s\n", status, name, detail)
	return err
}

func doctorOK(w io.Writer, name, detail string) error {
	return doctorLine(w, "OK", name, detail)
}

func doctorWarn(w io.Writer, stats *doctorStats, name, detail string) error {
	stats.warnings++
	return doctorLine(w, "WARN", name, detail)
}

func doctorFail(w io.Writer, stats *doctorStats, name, detail string) error {
	stats.failures++
	return doctorLine(w, "FAIL", name, detail)
}

func doctorFileError(label, path string, err error) string {
	if os.IsNotExist(err) {
		return fmt.Sprintf("missing %s: %s", label, path)
	}
	return err.Error()
}

func compilerVersionLine(compilerPath string) (string, error) {
	output, err := exec.Command(compilerPath, "--version").CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("%s", message)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return "version output is empty", nil
	}
	return strings.TrimSpace(lines[0]), nil
}

func cmdDoctor(root string, stdout io.Writer) error {
	stats := doctorStats{}

	if _, err := fmt.Fprintln(stdout, "cpx doctor"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "root: %s\n", root); err != nil {
		return err
	}

	appRoot := filepath.Join(root, appDir)
	workspaceReady := false
	if info, err := os.Stat(appRoot); err != nil {
		if err := doctorFail(stdout, &stats, "workspace", doctorFileError("workspace directory", appRoot, err)); err != nil {
			return err
		}
	} else if !info.IsDir() {
		if err := doctorFail(stdout, &stats, "workspace", fmt.Sprintf("expected a directory: %s", appRoot)); err != nil {
			return err
		}
	} else {
		workspaceReady = true
		if err := doctorOK(stdout, "workspace", appRoot); err != nil {
			return err
		}
	}

	if workspaceReady {
		configFile := filepath.Join(root, configPath)
		if cfg, err := readConfig(root); err != nil {
			if err := doctorFail(stdout, &stats, "config", err.Error()); err != nil {
				return err
			}
		} else {
			detail := fmt.Sprintf("%s exists; language=%s standard=%s", configFile, cfg.Language, cfg.Standard)
			if err := doctorOK(stdout, "config", detail); err != nil {
				return err
			}
		}

		templatesDir := filepath.Join(root, appDir, "templates")
		if info, err := os.Stat(templatesDir); err != nil {
			if err := doctorFail(stdout, &stats, "templates", doctorFileError("templates directory", templatesDir, err)); err != nil {
				return err
			}
		} else if !info.IsDir() {
			if err := doctorFail(stdout, &stats, "templates", fmt.Sprintf("expected a directory: %s", templatesDir)); err != nil {
				return err
			}
		} else if err := doctorOK(stdout, "templates", templatesDir); err != nil {
			return err
		}

		templateFile := filepath.Join(root, templatePath)
		if info, err := os.Stat(templateFile); err != nil {
			if err := doctorFail(stdout, &stats, "default template", doctorFileError("template", templateFile, err)); err != nil {
				return err
			}
		} else if info.IsDir() {
			if err := doctorFail(stdout, &stats, "default template", fmt.Sprintf("expected a file: %s", templateFile)); err != nil {
				return err
			}
		} else if err := doctorOK(stdout, "default template", templateFile); err != nil {
			return err
		}
	} else {
		if err := doctorWarn(stdout, &stats, "workspace setup", "run `cpx init` in this directory to create config and templates"); err != nil {
			return err
		}
	}

	if preferred := strings.TrimSpace(os.Getenv("CXX")); preferred != "" {
		if err := doctorOK(stdout, "CXX", preferred); err != nil {
			return err
		}
	} else if err := doctorWarn(stdout, &stats, "CXX", "not set; cpx will auto-detect a compiler from PATH"); err != nil {
		return err
	}

	compilerName, compilerPath, err := findCPPCompiler()
	if err != nil {
		if err := doctorFail(stdout, &stats, "compiler", err.Error()); err != nil {
			return err
		}
	} else {
		if err := doctorOK(stdout, "compiler", fmt.Sprintf("%s (%s)", compilerName, compilerPath)); err != nil {
			return err
		}

		versionLine, err := compilerVersionLine(compilerPath)
		if err != nil {
			if err := doctorFail(stdout, &stats, "compiler invocation", err.Error()); err != nil {
				return err
			}
		} else if err := doctorOK(stdout, "compiler version", versionLine); err != nil {
			return err
		}

		if runtime.GOOS == "windows" {
			if root, msystem, ok := detectMSYS2Compiler(compilerPath); ok {
				if err := doctorOK(stdout, "MSYS2", fmt.Sprintf("%s (%s)", root, msystem)); err != nil {
					return err
				}
			} else if err := doctorWarn(stdout, &stats, "MSYS2", "compiler is not from an MSYS2 MinGW environment"); err != nil {
				return err
			}
		}
	}

	summary := fmt.Sprintf("summary: %d failure(s), %d warning(s)", stats.failures, stats.warnings)
	if stats.failures == 0 {
		if err := doctorOK(stdout, "doctor", summary); err != nil {
			return err
		}
		return nil
	}

	if err := doctorLine(stdout, "FAIL", "doctor", summary); err != nil {
		return err
	}
	return fmt.Errorf("doctor found %d failure(s)", stats.failures)
}
