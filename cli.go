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

func printUsageError(stderr io.Writer, message, command string) {
	fmt.Fprintf(stderr, "Error: %s\n", message)
	fmt.Fprintf(stderr, "Usage: %s\n", usageFor(command))
}

func runCLI(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		printHelp(stdout)
		return 0
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}

	switch args[0] {
	case "version":
		if len(args) != 1 {
			printUsageError(stderr, "version does not accept arguments", "version")
			return 1
		}
		if err := cmdVersion(stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "doctor":
		if len(args) != 1 {
			printUsageError(stderr, "doctor does not accept arguments", "doctor")
			return 1
		}
		if err := cmdDoctor(cwd, stdout); err != nil {
			if errors.Is(err, errDoctorFailed) {
				return 1
			}
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "init":
		if len(args) != 1 {
			printUsageError(stderr, "init does not accept arguments", "init")
			return 1
		}
		if err := cmdInit(cwd, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "new":
		if len(args) < 2 || len(args) > 4 {
			printUsageError(stderr, "new requires a problem name, with optional sample count and template name", "new")
			return 1
		}
		options, err := parseNewOptions(args[2:])
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		if err := cmdNew(cwd, args[1], options, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "contest":
		if len(args) < 3 || len(args) > 5 {
			printUsageError(stderr, "contest requires a contest name, problem count, and optional sample count and template name", "contest")
			return 1
		}
		problemCount, err := parseContestProblemCount(args[2])
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		options, err := parseNewOptions(args[3:])
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		if err := cmdContest(cwd, args[1], problemCount, options, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "s":
		if len(args) < 2 || len(args) > 3 {
			printUsageError(stderr, "s requires a problem name and an optional sample count", "s")
			return 1
		}
		sampleCount, err := parseSampleCountArg(args[2:])
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		if err := cmdAddSamples(cwd, args[1], sampleCount, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "run":
		if len(args) != 2 {
			printUsageError(stderr, "run requires a problem name", "run")
			return 1
		}
		if err := cmdRun(cwd, args[1], stdout); err != nil {
			if errors.Is(err, errRunHandled) {
				return 1
			}
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "watch":
		if len(args) != 2 {
			printUsageError(stderr, "watch requires a problem name", "watch")
			return 1
		}
		if err := cmdWatch(cwd, args[1], stdout); err != nil {
			if errors.Is(err, errRunHandled) {
				return 1
			}
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "Error: unknown command")
		return 1
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: cpx [command]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  version                 print cpx version")
	fmt.Fprintln(w, "  doctor                  check local cpx setup")
	fmt.Fprintln(w, "  init                    initialize competitive programming workspace")
	fmt.Fprintln(w, "  new <problem> [count] [template] create a new problem folder")
	fmt.Fprintln(w, "  contest <contest> <count> [samples] [template] create a contest folder")
	fmt.Fprintln(w, "  s <problem> [count]     add sample files to a problem")
	fmt.Fprintln(w, "  run <problem>           compile and test a problem")
	fmt.Fprintln(w, "  watch <problem>         rerun a problem when files change")
}
