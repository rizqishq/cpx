package main

import (
	"fmt"
	"io"
	"os"
)

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
			fmt.Fprintln(stderr, "Error: init does not accept arguments")
			return 1
		}
		if err := cmdInit(cwd, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "new":
		if len(args) < 2 || len(args) > 3 {
			fmt.Fprintln(stderr, "Error: new requires a problem name and an optional sample count")
			return 1
		}
		sampleCount, err := parseSampleCountArg(args[2:])
		if err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		if err := cmdNew(cwd, args[1], sampleCount, stdout); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}
		return 0
	case "s":
		if len(args) < 2 || len(args) > 3 {
			fmt.Fprintln(stderr, "Error: s requires a problem name and an optional sample count")
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
			fmt.Fprintln(stderr, "Error: run requires a problem name")
			return 1
		}
		if err := cmdRun(cwd, args[1], stdout); err != nil {
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
	fmt.Fprintln(w, "  init          initialize competitive programming workspace")
	fmt.Fprintln(w, "  new <problem> [count] create a new problem folder")
	fmt.Fprintln(w, "  s <problem> [count]   add sample files to a problem")
	fmt.Fprintln(w, "  run <problem>         compile and test a problem")
}
