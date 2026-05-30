package main

import (
	"fmt"
	"io"
	"os"
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
			fmt.Fprintf(stderr, "Error: init does not accept arguments\n\nusage: cpx init\n")
			return 1
		}
		if err := cmdInit(cwd, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "new":
		if len(args) < 2 || len(args) > 4 {
			fmt.Fprintf(stderr, "Error: invalid arguments for new\n\nusage: cpx new <problem> [count] [template]\nexample:\n  cpx new a\n  cpx new b 3\n  cpx new c debug\n  cpx new d 2 debug\n")
			return 1
		}
		sampleCount, templateName, err := parseNewArgs(args[2:])
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n\nusage: cpx new <problem> [count] [template]\n", err)
			return 1
		}
		if err := cmdNew(cwd, args[1], sampleCount, templateName, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "contest":
		problems, err := parseContestArgs(args[1:])
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n\nusage: cpx contest <problem>...\nexample:\n  cpx contest a b c d\n", err)
			return 1
		}
		if err := cmdContest(cwd, problems, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "s":
		if len(args) < 2 || len(args) > 3 {
			fmt.Fprintf(stderr, "Error: invalid arguments for s\n\nusage: cpx s <problem> [count]\nexample:\n  cpx s a\n  cpx s a 2\n")
			return 1
		}
		sampleCount, err := parseSampleCountArg(args[2:])
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n\nusage: cpx s <problem> [count]\n", err)
			return 1
		}
		if err := cmdAddSamples(cwd, args[1], sampleCount, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "run":
		if len(args) != 2 {
			fmt.Fprintf(stderr, "Error: run requires exactly one problem name\n\nusage: cpx run <problem>\nexample:\n  cpx run a\n")
			return 1
		}
		if err := cmdRun(cwd, args[1], stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "doctor":
		if len(args) != 1 {
			fmt.Fprintf(stderr, "Error: doctor does not accept arguments\n\nusage: cpx doctor\n")
			return 1
		}
		if err := cmdDoctor(cwd, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "version":
		if len(args) != 1 {
			fmt.Fprintf(stderr, "Error: version does not accept arguments\n\nusage: cpx version\n")
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

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "usage: cpx [command]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  init                             initialize competitive programming workspace")
	fmt.Fprintln(w, "  new <problem> [count] [template] create a new problem folder")
	fmt.Fprintln(w, "  contest <problem>...             create multiple problem folders at once")
	fmt.Fprintln(w, "  s <problem> [count]              add sample files to a problem")
	fmt.Fprintln(w, "  run <problem>                    compile and test a problem")
	fmt.Fprintln(w, "  doctor                           check workspace and compiler setup")
	fmt.Fprintln(w, "  version                          print the installed cpx version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "examples:")
	fmt.Fprintln(w, "  cpx init")
	fmt.Fprintln(w, "  cpx new a")
	fmt.Fprintln(w, "  cpx contest a b c d")
	fmt.Fprintln(w, "  cpx new b 3")
	fmt.Fprintln(w, "  cpx new c debug")
	fmt.Fprintln(w, "  cpx doctor")
	fmt.Fprintln(w, "  cpx run a")
}
