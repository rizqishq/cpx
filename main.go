package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var version = "dev"

const (
	appDir       = ".cpx"
	configPath   = ".cpx/config.json"
	templatePath = ".cpx/templates/main.cpp"
)

const legacyDefaultTemplate = `#include <bits/stdc++.h>
using namespace std;

int main() {
    ios::sync_with_stdio(false);
    cin.tie(nullptr);

    return 0;
}
`

const defaultTemplate = `#include <iostream>
using namespace std;

int main() {
    ios::sync_with_stdio(false);
    cin.tie(nullptr);

    return 0;
}
`

type config struct {
	Language string `json:"language"`
	Standard string `json:"standard"`
}

var defaultWorkspaceConfig = config{
	Language: "cpp",
	Standard: "c++17",
}

const (
	usageInit    = "usage: cpx init"
	usageNew     = "usage: cpx new <problem> [count] [template]"
	usageContest = "usage: cpx contest <problem>... [-c <count>] [-t <template>]"
	usageSample  = "usage: cpx s <problem> [count]"
	usageRun     = "usage: cpx run <problem>"
	usageWatch   = "usage: cpx watch <problem>"
	usageDoctor  = "usage: cpx doctor"
	usageVersion = "usage: cpx version"
)

var helpExamples = []string{
	"cpx init",
	"cpx new a",
	"cpx contest a b c d",
	"cpx contest a b c d -c 3",
	"cpx contest a b c d -t debug",
	"cpx new b 3",
	"cpx new c debug",
	"cpx watch a",
	"cpx doctor",
	"cpx run a",
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
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
	case "init":
		if len(args) != 1 {
			printErrorUsage(stderr, "init does not accept arguments", usageInit)
			return 1
		}
		if err := cmdInit(cwd, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "new":
		if len(args) < 2 || len(args) > 4 {
			printErrorUsage(stderr, "invalid arguments for new", usageNew, []string{"cpx new a", "cpx new b 3", "cpx new c debug", "cpx new d 2 debug"}...)
			return 1
		}
		sampleCount, templateName, err := parseNewArgs(args[2:])
		if err != nil {
			printErrorUsage(stderr, err.Error(), usageNew)
			return 1
		}
		if err := cmdNew(cwd, args[1], sampleCount, templateName, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "contest":
		problems, sampleCount, templateName, err := parseContestArgs(args[1:])
		if err != nil {
			printErrorUsage(stderr, err.Error(), usageContest, "cpx contest a b c d", "cpx contest a b c d -c 3", "cpx contest a b c d --template debug", "cpx contest a b c d -c 2 -t debug")
			return 1
		}
		if err := cmdContest(cwd, problems, sampleCount, templateName, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "s":
		if len(args) < 2 || len(args) > 3 {
			printErrorUsage(stderr, "invalid arguments for s", usageSample, "cpx s a", "cpx s a 2")
			return 1
		}
		sampleCount, err := parseSampleCountArg(args[2:])
		if err != nil {
			printErrorUsage(stderr, err.Error(), usageSample)
			return 1
		}
		if err := cmdAddSamples(cwd, args[1], sampleCount, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "run":
		if len(args) != 2 {
			printErrorUsage(stderr, "run requires exactly one problem name", usageRun, "cpx run a")
			return 1
		}
		if err := cmdRun(cwd, args[1], stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "watch":
		if len(args) != 2 {
			printErrorUsage(stderr, "watch requires exactly one problem name", usageWatch, "cpx watch a")
			return 1
		}
		if err := cmdWatch(cwd, args[1], stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "doctor":
		if len(args) != 1 {
			printErrorUsage(stderr, "doctor does not accept arguments", usageDoctor)
			return 1
		}
		if err := cmdDoctor(cwd, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "version":
		if len(args) != 1 {
			printErrorUsage(stderr, "version does not accept arguments", usageVersion)
			return 1
		}
		if _, err := fmt.Fprintf(stdout, "cpx %s\n", version); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "Error: unknown command %q\n\nRun `cpx help` to see available commands.\n", args[0])
		return 1
	}
}

func printErrorUsage(w io.Writer, message, usage string, examples ...string) {
	fmt.Fprintf(w, "Error: %s\n\n%s\n", message, usage)
	if len(examples) == 0 {
		return
	}
	label := "examples"
	if len(examples) == 1 {
		label = "example"
	}
	fmt.Fprintf(w, "%s:\n  %s\n", label, strings.Join(examples, "\n  "))
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: cpx [command]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  init                             initialize competitive programming workspace")
	fmt.Fprintln(w, "  new <problem> [count] [template] create a new problem folder")
	fmt.Fprintln(w, "  contest <problem>... [-c count] [-t template] create multiple problem folders at once")
	fmt.Fprintln(w, "  s <problem> [count]              add sample files to a problem")
	fmt.Fprintln(w, "  run <problem>                    compile and test a problem")
	fmt.Fprintln(w, "  watch <problem>                  rerun a problem whenever source or samples change")
	fmt.Fprintln(w, "  doctor                           check workspace and compiler setup")
	fmt.Fprintln(w, "  version                          print the installed cpx version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "examples:")
	for _, example := range helpExamples {
		fmt.Fprintf(w, "  %s\n", example)
	}
}
