package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type doctorStatus string

const (
	doctorOK   doctorStatus = "OK"
	doctorWarn doctorStatus = "WARN"
	doctorFail doctorStatus = "FAIL"
)

var errDoctorFailed = errors.New("doctor found one or more problems")

type doctorCheck struct {
	status doctorStatus
	label  string
	detail string
}

func (c doctorCheck) writeTo(w io.Writer) error {
	_, err := fmt.Fprintf(w, "[%s] %s: %s\n", colorizeDoctorStatus(c.status), c.label, c.detail)
	return err
}

type doctorResult struct {
	checks []doctorCheck
	ok     int
	warn   int
	fail   int
}

func (r *doctorResult) add(status doctorStatus, label, detail string) {
	r.checks = append(r.checks, doctorCheck{status: status, label: label, detail: detail})
	switch status {
	case doctorOK:
		r.ok++
	case doctorWarn:
		r.warn++
	case doctorFail:
		r.fail++
	}
}

func cmdDoctor(root string, stdout io.Writer) error {
	result := collectDoctorChecks(root)
	for _, check := range result.checks {
		if err := check.writeTo(stdout); err != nil {
			return err
		}
	}

	summary := fmt.Sprintf("%d OK, %d WARN, %d FAIL", result.ok, result.warn, result.fail)
	if _, err := fmt.Fprintf(stdout, "Summary: %s\n", summary); err != nil {
		return err
	}

	if result.fail > 0 {
		return errDoctorFailed
	}
	return nil
}

func collectDoctorChecks(root string) doctorResult {
	var result doctorResult

	result.add(doctorOK, "version", resolvedVersion())

	if exePath, err := os.Executable(); err != nil {
		result.add(doctorWarn, "binary", fmt.Sprintf("resolve executable path: %v", err))
	} else {
		result.add(doctorOK, "binary", exePath)
	}

	if commandPath, err := exec.LookPath("cpx"); err != nil {
		result.add(doctorWarn, "command path", "cpx not found in PATH")
	} else {
		result.add(doctorOK, "command path", commandPath)
	}

	result.add(doctorOK, "os", runtime.GOOS)
	result.add(doctorOK, "arch", runtime.GOARCH)
	result.add(doctorOK, "cwd", root)

	defaultSpec, toolName, toolPath, toolErr := resolveRuntimeToolForConfig(defaultConfig())
	defaultToolLabel := defaultSpec.toolLabel
	toolReady := toolErr == nil

	configFile := filepath.Join(root, configPath)
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if defaultToolLabel == "compiler" {
			preferredCompiler := strings.TrimSpace(os.Getenv("CXX"))
			if preferredCompiler == "" {
				result.add(doctorOK, "CXX", "not set")
			} else {
				result.add(doctorOK, "CXX", preferredCompiler)
			}
		}
		if toolErr != nil {
			result.add(doctorFail, defaultToolLabel, toolErr.Error())
		} else {
			result.add(doctorOK, defaultToolLabel, fmt.Sprintf("%s (%s)", toolName, toolPath))
			toolVersion := runtimeToolVersionCheck(defaultToolLabel, toolPath)
			result.add(toolVersion.status, toolVersion.label, toolVersion.detail)
		}
		result.add(doctorWarn, "workspace", "not initialized; run 'cpx init' first")
		if toolReady {
			result.add(doctorWarn, "run readiness", fmt.Sprintf("%s is ready, but workspace is not initialized", defaultToolLabel))
		} else {
			result.add(doctorFail, "run readiness", fmt.Sprintf("%s is missing and workspace is not initialized", defaultToolLabel))
		}
		return result
	} else if err != nil {
		result.add(doctorFail, "workspace", fmt.Sprintf("check %s: %v", configFile, err))
		result.add(doctorFail, "run readiness", "workspace check failed")
		return result
	}

	workspaceDir := filepath.Join(root, appDir)
	result.add(doctorOK, "workspace", workspaceDir)

	cfg, err := loadConfig(root)
	if err != nil {
		result.add(doctorFail, "config", err.Error())
		result.add(doctorFail, "run readiness", "config is invalid")
		return result
	}

	result.add(doctorOK, "config language", cfg.Language)
	result.add(doctorOK, "config standard", cfg.Standard)
	result.add(doctorOK, "config template", cfg.Template)
	result.add(doctorOK, "config runTimeoutMs", fmt.Sprintf("%d", cfg.RunTimeoutMs))
	result.add(doctorOK, "config stopOnFirstFail", fmt.Sprintf("%t", stopOnFirstFailEnabled(cfg)))
	result.add(doctorOK, "config diffContextLines", fmt.Sprintf("%d", cfg.DiffContextLines))
	result.add(doctorOK, "config watchIntervalMs", fmt.Sprintf("%d", cfg.WatchIntervalMs))
	if len(cfg.CompilerFlags) > 0 {
		result.add(doctorOK, "config compilerFlags", strings.Join(cfg.CompilerFlags, " "))
	}

	spec, toolName, toolPath, toolErr := resolveRuntimeToolForConfig(cfg)
	toolLabel := spec.toolLabel
	toolReady = toolErr == nil
	if toolLabel == "compiler" {
		preferredCompiler := strings.TrimSpace(os.Getenv("CXX"))
		if preferredCompiler == "" {
			result.add(doctorOK, "CXX", "not set")
		} else {
			result.add(doctorOK, "CXX", preferredCompiler)
		}
	}
	if toolErr != nil {
		result.add(doctorFail, toolLabel, toolErr.Error())
	} else {
		result.add(doctorOK, toolLabel, fmt.Sprintf("%s (%s)", toolName, toolPath))
		toolVersion := runtimeToolVersionCheck(toolLabel, toolPath)
		result.add(toolVersion.status, toolVersion.label, toolVersion.detail)
	}

	available, err := availableTemplates(root, cfg)
	if err != nil {
		result.add(doctorFail, "templates", err.Error())
		result.add(doctorFail, "run readiness", "template discovery failed")
		return result
	}

	if len(available) == 0 {
		result.add(doctorFail, "templates", "no templates found in .cpx/templates")
		result.add(doctorFail, "run readiness", "no templates are available")
		return result
	}

	result.add(doctorOK, "templates count", fmt.Sprintf("%d", len(available)))
	result.add(doctorOK, "templates", strings.Join(available, ", "))

	templateRelPath, err := templateRelativePath(cfg, cfg.Template)
	if err != nil {
		result.add(doctorFail, "default template", err.Error())
		result.add(doctorFail, "run readiness", "default template is invalid")
		return result
	}

	if _, err := readTemplate(root, cfg, cfg.Template); err != nil {
		result.add(doctorFail, "default template", err.Error())
		result.add(doctorFail, "run readiness", "default template is missing")
		return result
	}

	result.add(doctorOK, "default template", cfg.Template)
	result.add(doctorOK, "default template file", filepath.Join(root, templateRelPath))

	if toolReady {
		result.add(doctorOK, "run readiness", "ready")
	} else {
		result.add(doctorFail, "run readiness", fmt.Sprintf("%s is missing", toolLabel))
	}

	return result
}

func runtimeToolVersionCheck(toolLabel, toolPath string) doctorCheck {
	cmd := exec.Command(toolPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return doctorCheck{
			status: doctorWarn,
			label:  toolLabel + " version",
			detail: fmt.Sprintf("run %s --version: %v", toolPath, err),
		}
	}

	firstLine := strings.TrimSpace(string(output))
	if newline := strings.IndexByte(firstLine, '\n'); newline >= 0 {
		firstLine = firstLine[:newline]
	}
	if firstLine == "" {
		firstLine = filepath.Base(toolPath)
	}

	return doctorCheck{
		status: doctorOK,
		label:  toolLabel + " version",
		detail: firstLine,
	}
}
