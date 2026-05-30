package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const watchInterval = 500 * time.Millisecond

type watchState struct {
	signature string
	changes   []string
}

func collectWatchEntries(problemDir string, cfg config) ([]string, error) {
	entries := []string{filepath.Join(problemDir, sourceFileName(cfg))}

	samplesDir := filepath.Join(problemDir, "samples")
	sampleEntries, err := os.ReadDir(samplesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return entries, nil
		}
		return nil, err
	}

	for _, entry := range sampleEntries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".in") && !strings.HasSuffix(name, ".out") {
			continue
		}
		entries = append(entries, filepath.Join(samplesDir, name))
	}

	sort.Strings(entries)
	return entries, nil
}

func buildWatchState(problemDir string, cfg config, previous map[string]string) (watchState, map[string]string, error) {
	paths, err := collectWatchEntries(problemDir, cfg)
	if err != nil {
		return watchState{}, nil, err
	}

	next := make(map[string]string, len(paths))
	signatureParts := make([]string, 0, len(paths))
	changes := make([]string, 0)

	for _, path := range paths {
		value := path + "|missing"
		info, err := os.Stat(path)
		if err == nil {
			value = fmt.Sprintf("%s|%d|%d", path, info.Size(), info.ModTime().UnixNano())
		} else if !os.IsNotExist(err) {
			return watchState{}, nil, err
		}

		next[path] = value
		signatureParts = append(signatureParts, value)

		if previous == nil {
			continue
		}
		if previousValue, ok := previous[path]; !ok {
			changes = append(changes, path)
		} else if previousValue != value {
			changes = append(changes, path)
		}
	}

	if previous != nil {
		for path := range previous {
			if _, ok := next[path]; !ok {
				changes = append(changes, path)
			}
		}
	}

	sort.Strings(changes)
	return watchState{
		signature: strings.Join(signatureParts, "\n"),
		changes:   changes,
	}, next, nil
}

func runWatchedProblem(root, problem string, stdout io.Writer) {
	if err := cmdRun(root, problem, stdout); err != nil {
		fmt.Fprintf(stdout, "Error: %v\n", err)
	}
}

func cmdWatch(root, problem string, stdout io.Writer) error {
	if err := validateProblemName(problem); err != nil {
		return err
	}

	cfg, err := readConfig(root)
	if err != nil {
		return err
	}

	problemDir := filepath.Join(root, problem)
	if _, err := fmt.Fprintf(stdout, "Watching %s for changes. Press Ctrl+C to stop.\n\n", problemDir); err != nil {
		return err
	}

	runWatchedProblem(root, problem, stdout)

	state, snapshot, err := buildWatchState(problemDir, cfg, nil)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	for {
		select {
		case <-signals:
			_, err := fmt.Fprintln(stdout, "\nStopped watching.")
			return err
		case <-ticker.C:
			nextState, nextSnapshot, err := buildWatchState(problemDir, cfg, snapshot)
			if err != nil {
				return err
			}
			snapshot = nextSnapshot
			if nextState.signature == state.signature {
				continue
			}
			state = nextState

			if len(nextState.changes) > 0 {
				if _, err := fmt.Fprintln(stdout); err != nil {
					return err
				}
				for _, path := range nextState.changes {
					if _, err := fmt.Fprintf(stdout, "Change detected: %s\n", path); err != nil {
						return err
					}
				}
				if _, err := fmt.Fprintln(stdout); err != nil {
					return err
				}
			}

			runWatchedProblem(root, problem, stdout)
		}
	}
}
