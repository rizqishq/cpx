package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type watchSnapshot map[string]time.Time

func cmdWatch(root, problem string, stdout io.Writer) error {
	if err := validateProblemPath(problem); err != nil {
		return err
	}

	snapshot, err := currentWatchSnapshot(root, problem)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(stdout, "Watching %s\n", problem); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "Watching files:"); err != nil {
		return err
	}
	for _, path := range sortedWatchPaths(snapshot) {
		if _, err := fmt.Fprintf(stdout, "  - %s\n", path); err != nil {
			return err
		}
	}

	runOnce := func(reason string) error {
		if _, err := fmt.Fprintln(stdout, strings.Repeat("=", 40)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(stdout, "[%s] %s\n", time.Now().Format("15:04:05"), reason); err != nil {
			return err
		}
		if err := cmdRun(root, problem, stdout); err != nil {
			if errors.Is(err, errRunHandled) {
				return nil
			}
			return err
		}
		return nil
	}

	if err := runOnce("Initial run"); err != nil {
		return err
	}

	cfg, err := loadConfig(root)
	if err != nil {
		return err
	}
	watchInterval := time.Duration(cfg.WatchIntervalMs) * time.Millisecond

	for {
		time.Sleep(watchInterval)

		nextCfg, cfgErr := loadConfig(root)
		if cfgErr == nil {
			watchInterval = time.Duration(nextCfg.WatchIntervalMs) * time.Millisecond
		}

		nextSnapshot, err := currentWatchSnapshot(root, problem)
		if err != nil {
			if _, writeErr := fmt.Fprintf(stdout, "%s failed to refresh watch snapshot: %v\n", colorizeWarningLabel(), err); writeErr != nil {
				return writeErr
			}
			continue
		}
		if !watchSnapshotsEqual(snapshot, nextSnapshot) {
			changed := diffWatchSnapshots(snapshot, nextSnapshot)
			if _, err := fmt.Fprintln(stdout, "Change detected:"); err != nil {
				return err
			}
			for _, path := range changed {
				if _, err := fmt.Fprintf(stdout, "  - %s\n", path); err != nil {
					return err
				}
			}
			if err := runOnce("Rerun after change"); err != nil {
				return err
			}
			snapshot = nextSnapshot
		}
	}
}

func currentWatchSnapshot(root, problem string) (watchSnapshot, error) {
	cfgPath := filepath.Join(root, configPath)
	cfg, err := loadConfig(root)
	if err != nil {
		return nil, err
	}

	sourceName, err := sourceFileName(cfg)
	if err != nil {
		return nil, err
	}

	snapshot := watchSnapshot{}
	paths := []string{cfgPath, filepath.Join(root, problem, sourceName)}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		snapshot[path] = info.ModTime()
	}

	samplesDir := filepath.Join(root, problem, "samples")
	entries, err := os.ReadDir(samplesDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(samplesDir, entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		snapshot[path] = info.ModTime()
	}

	return snapshot, nil
}

func watchSnapshotsEqual(left, right watchSnapshot) bool {
	if len(left) != len(right) {
		return false
	}

	for path, leftTime := range left {
		rightTime, ok := right[path]
		if !ok || !leftTime.Equal(rightTime) {
			return false
		}
	}
	return true
}

func sortedWatchPaths(snapshot watchSnapshot) []string {
	paths := make([]string, 0, len(snapshot))
	for path := range snapshot {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func diffWatchSnapshots(left, right watchSnapshot) []string {
	changed := make([]string, 0)
	seen := make(map[string]struct{})

	for path, leftTime := range left {
		rightTime, ok := right[path]
		if !ok || !leftTime.Equal(rightTime) {
			changed = append(changed, path)
			seen[path] = struct{}{}
		}
	}
	for path := range right {
		if _, ok := left[path]; ok {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		changed = append(changed, path)
	}
	sort.Strings(changed)
	return changed
}
