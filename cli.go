package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func usageFor(command string) string {
	switch command {
	case "version":
		return "cpx version"
	case "doctor":
		return "cpx doctor"
	case "init":
		return "cpx init"
	case "new":
		return "cpx new <problem> [count] [template]"
	case "contest":
		return "cpx contest <contest> <count> [samples] [template]"
	case "s":
		return "cpx s <problem> [count]"
	case "run":
		return "cpx run <problem>"
	case "watch":
		return "cpx watch <problem>"
	default:
		return "cpx [command]"
	}
}

func helpEntries() []commandHelpEntry {
	return []commandHelpEntry{
		{command: "version", description: "print cpx version"},
		{command: "doctor", description: "check local cpx setup"},
		{command: "init", description: "initialize competitive programming workspace"},
		{command: "new <problem> [count] [template]", description: "create a new problem folder"},
		{command: "contest <contest> <count> [samples] [template]", description: "create a contest folder"},
		{command: "s <problem> [count]", description: "add sample files to a problem"},
		{command: "run <problem>", description: "compile and test a problem"},
		{command: "watch <problem>", description: "rerun a problem when files change"},
	}
}

func descriptionFor(command string) (string, bool) {
	switch command {
	case "version":
		return "print cpx version", true
	case "doctor":
		return "check local cpx setup", true
	case "init":
		return "initialize competitive programming workspace", true
	case "new":
		return "create a new problem folder", true
	case "contest":
		return "create a contest folder", true
	case "s":
		return "add sample files to a problem", true
	case "run":
		return "compile and test a problem", true
	case "watch":
		return "rerun a problem when files change", true
	default:
		return "", false
	}
}

func isHelpToken(value string) bool {
	return value == "-h" || value == "--help" || value == "help"
}

func printCommandHelp(w io.Writer, command string) bool {
	description, ok := descriptionFor(command)
	if !ok {
		return false
	}
	fmt.Fprintf(w, "usage: %s\n", usageFor(command))
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s\n", description)
	return true
}

func printUsageError(stderr io.Writer, message, command string) {
	fmt.Fprintf(stderr, "%s %s\n", colorizeErrorLabel(), message)
	fmt.Fprintf(stderr, "Usage: %s\n", usageFor(command))
}

func printUnknownCommandError(stderr io.Writer, command string) {
	fmt.Fprintf(stderr, "%s unknown command: %s\n", colorizeErrorLabel(), command)
	fmt.Fprintf(stderr, "Usage: %s\n", usageFor(""))
	fmt.Fprintln(stderr, "Run `cpx help` to see available commands.")
}

func runCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	configureColorSupport(stdout, stderr)

	if len(args) == 0 || isHelpToken(args[0]) {
		if len(args) <= 1 {
			printHelp(stdout)
			return 0
		}
		if printCommandHelp(stdout, args[1]) {
			return 0
		}
		printUnknownCommandError(stderr, args[1])
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
		return 1
	}

	switch args[0] {
	case "version":
		if len(args) == 2 && isHelpToken(args[1]) {
			printCommandHelp(stdout, "version")
			return 0
		}
		if len(args) != 1 {
			printUsageError(stderr, "version does not accept arguments", "version")
			return 1
		}
		if err := cmdVersion(stdout); err != nil {
			fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
			return 1
		}
		return 0
	case "doctor":
		if len(args) == 2 && isHelpToken(args[1]) {
			printCommandHelp(stdout, "doctor")
			return 0
		}
		if len(args) != 1 {
			printUsageError(stderr, "doctor does not accept arguments", "doctor")
			return 1
		}
		if err := cmdDoctor(cwd, stdout); err != nil {
			if errors.Is(err, errDoctorFailed) {
				return 1
			}
			fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
			return 1
		}
		return 0
	case "init":
		if len(args) == 2 && isHelpToken(args[1]) {
			printCommandHelp(stdout, "init")
			return 0
		}
		if len(args) != 1 {
			printUsageError(stderr, "init does not accept arguments", "init")
			return 1
		}
		if err := cmdInit(cwd, stdout); err != nil {
			fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
			return 1
		}
		return 0
	case "new":
		if len(args) == 2 && isHelpToken(args[1]) {
			printCommandHelp(stdout, "new")
			return 0
		}
		if len(args) < 2 || len(args) > 4 {
			printUsageError(stderr, "new requires a problem name, with optional sample count and template name", "new")
			return 1
		}
		options, err := parseNewOptions(args[2:])
		if err != nil {
			fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
			return 1
		}
		if err := cmdNew(cwd, args[1], options, stdout); err != nil {
			fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
			return 1
		}
		return 0
	case "contest":
		if len(args) == 2 && isHelpToken(args[1]) {
			printCommandHelp(stdout, "contest")
			return 0
		}
		if len(args) < 3 || len(args) > 5 {
			printUsageError(stderr, "contest requires a contest name, problem count, and optional sample count and template name", "contest")
			return 1
		}
		problemCount, err := parseContestProblemCount(args[2])
		if err != nil {
			fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
			return 1
		}
		options, err := parseNewOptions(args[3:])
		if err != nil {
			fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
			return 1
		}
		if err := cmdContest(cwd, args[1], problemCount, options, stdout); err != nil {
			fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
			return 1
		}
		return 0
	case "s":
		if len(args) == 2 && isHelpToken(args[1]) {
			printCommandHelp(stdout, "s")
			return 0
		}
		if len(args) < 2 || len(args) > 3 {
			printUsageError(stderr, "s requires a problem name and an optional sample count", "s")
			return 1
		}
		sampleCount, err := parseSampleCountArg(args[2:])
		if err != nil {
			fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
			return 1
		}
		if err := cmdAddSamples(cwd, args[1], sampleCount, stdout); err != nil {
			fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
			return 1
		}
		return 0
	case "run":
		if len(args) == 2 && isHelpToken(args[1]) {
			printCommandHelp(stdout, "run")
			return 0
		}
		if len(args) != 2 {
			printUsageError(stderr, "run requires a problem name", "run")
			return 1
		}
		if err := cmdRun(cwd, args[1], stdout); err != nil {
			if errors.Is(err, errRunHandled) {
				return 1
			}
			fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
			return 1
		}
		return 0
	case "watch":
		if len(args) == 2 && isHelpToken(args[1]) {
			printCommandHelp(stdout, "watch")
			return 0
		}
		if len(args) != 2 {
			printUsageError(stderr, "watch requires a problem name", "watch")
			return 1
		}
		if err := cmdWatch(cwd, args[1], stdout); err != nil {
			if errors.Is(err, errRunHandled) {
				return 1
			}
			fmt.Fprintf(stderr, "%s %v\n", colorizeErrorLabel(), err)
			return 1
		}
		return 0
	default:
		printUnknownCommandError(stderr, args[0])
		return 1
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: cpx [command]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	_ = writeAlignedCommandTable(w, helpEntries())
}
