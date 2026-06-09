package main

const (
	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
)

func colorize(text, color string) string {
	return color + text + ansiReset
}

func colorizeDoctorStatus(status doctorStatus) string {
	switch status {
	case doctorOK:
		return colorize(string(status), ansiGreen)
	case doctorWarn:
		return colorize(string(status), ansiYellow)
	case doctorFail:
		return colorize(string(status), ansiRed)
	default:
		return string(status)
	}
}

func colorizeRunStatus(status string) string {
	switch status {
	case "PASS":
		return colorize(status, ansiGreen)
	case "FAIL":
		return colorize(status, ansiRed)
	default:
		return status
	}
}

func colorizeErrorLabel() string {
	return colorize("Error:", ansiRed)
}

func colorizeWarningLabel() string {
	return colorize("Warning:", ansiYellow)
}
