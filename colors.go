package main

import (
	"io"
	"os"
)

const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
)

var (
	stdoutColorsEnabled bool
	stderrColorsEnabled bool
)

func configureColorSupport(stdout, stderr io.Writer) {
	stdoutColorsEnabled = outputSupportsColor(stdout)
	stderrColorsEnabled = outputSupportsColor(stderr)
}

func outputSupportsColor(w io.Writer) bool {
	if _, disabled := os.LookupEnv("NO_COLOR"); disabled {
		return false
	}

	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func colorize(text, color string, enabled bool) string {
	if !enabled {
		return text
	}
	return color + text + ansiReset
}

func colorizeDoctorStatus(status doctorStatus) string {
	switch status {
	case doctorOK:
		return colorize(string(status), ansiGreen, stdoutColorsEnabled)
	case doctorWarn:
		return colorize(string(status), ansiYellow, stdoutColorsEnabled)
	case doctorFail:
		return colorize(string(status), ansiRed, stdoutColorsEnabled)
	default:
		return string(status)
	}
}

func colorizeRunStatus(status string) string {
	switch status {
	case "PASS":
		return colorize(status, ansiGreen, stdoutColorsEnabled)
	case "FAIL":
		return colorize(status, ansiRed, stdoutColorsEnabled)
	default:
		return status
	}
}

func colorizeErrorLabel() string {
	return colorize("Error:", ansiRed, stderrColorsEnabled)
}

func colorizeWarningLabel() string {
	return colorize("Warning:", ansiYellow, stdoutColorsEnabled)
}
