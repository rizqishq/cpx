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

func (c doctorCheck) writeTo(w io.Writer, labelWidth int) error {
	_, err := fmt.Fprintf(w, "[%s] %s\n", colorizeDoctorStatus(c.status), formatAlignedField(c.label, c.detail, labelWidth))
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
	labels := make([]string, 0, len(result.checks))
	for _, check := range result.checks {
		labels = append(labels, check.label)
	}
	labelWidth := maxWidth(labels) + 1

	for _, check := range result.checks {
		if err := check.writeTo(stdout, labelWidth); err != nil {
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

	preferredCompiler := strings.TrimSpace(os.Getenv("CXX"))
	if preferredCompiler == "" {
		result.add(doctorOK, "CXX", "not set")
	} else {
		result.add(doctorOK, "CXX", preferredCompiler)
	}

	compilerName, compilerPath, compilerErr := findCPPCompiler()
	compilerReady := compilerErr == nil
	if compilerErr != nil {
		result.add(doctorFail, "compiler", compilerErr.Error())
	} else {
		result.add(doctorOK, "compiler", fmt.Sprintf("%s (%s)", compilerName, compilerPath))
		compilerVersion := compilerVersionCheck(compilerPath)
		result.add(compilerVersion.status, compilerVersion.label, compilerVersion.detail)
	}

	configFile := filepath.Join(root, configPath)
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		result.add(doctorWarn, "workspace", "not initialized; run 'cpx init' first")
		if compilerReady {
			result.add(doctorWarn, "run readiness", "compiler is ready, but workspace is not initialized")
		} else {
			result.add(doctorFail, "run readiness", "compiler is missing and workspace is not initialized")
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
	if len(cfg.CompilerFlags) > 0 {
		result.add(doctorOK, "config compilerFlags", strings.Join(cfg.CompilerFlags, " "))
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

	if compilerReady {
		result.add(doctorOK, "run readiness", "ready")
	} else {
		result.add(doctorFail, "run readiness", "compiler is missing")
	}

	return result
}

func compilerVersionCheck(compilerPath string) doctorCheck {
	cmd := exec.Command(compilerPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return doctorCheck{
			status: doctorWarn,
			label:  "compiler version",
			detail: fmt.Sprintf("run %s --version: %v", compilerPath, err),
		}
	}

	firstLine := strings.TrimSpace(string(output))
	if newline := strings.IndexByte(firstLine, '\n'); newline >= 0 {
		firstLine = firstLine[:newline]
	}
	if firstLine == "" {
		firstLine = filepath.Base(compilerPath)
	}

	return doctorCheck{
		status: doctorOK,
		label:  "compiler version",
		detail: firstLine,
	}
}
