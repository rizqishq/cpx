package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func cmdInit(root string, stdout io.Writer) error {
	if err := ensureWorkspace(root); err != nil {
		return err
	}
	_, err := fmt.Fprintf(stdout, "Initialized workspace at %s\n", filepath.Join(root, appDir))
	return err
}

func parseSampleCountArg(args []string) (int, error) {
	if len(args) == 0 {
		return 1, nil
	}

	count, err := strconv.Atoi(args[0])
	if err != nil || count < 1 {
		return 0, fmt.Errorf("sample count must be a positive integer")
	}
	return count, nil
}

func createSampleFiles(samplesDir string, start, count int) error {
	for index := 0; index < count; index++ {
		sampleNumber := start + index
		inputPath := filepath.Join(samplesDir, fmt.Sprintf("%d.in", sampleNumber))
		outputPath := filepath.Join(samplesDir, fmt.Sprintf("%d.out", sampleNumber))
		if err := os.WriteFile(inputPath, []byte{}, 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(outputPath, []byte{}, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func nextSampleNumber(samplesDir string) (int, error) {
	entries, err := os.ReadDir(samplesDir)
	if err != nil {
		return 0, err
	}

	maxSample := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".in") {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), ".in")
		number, err := strconv.Atoi(base)
		if err != nil {
			continue
		}
		if number > maxSample {
			maxSample = number
		}
	}
	return maxSample + 1, nil
}

func cmdNew(root, problem string, sampleCount int, stdout io.Writer) error {
	cfg, err := loadConfig(root)
	if err != nil {
		return err
	}

	template, err := readTemplate(root, cfg)
	if err != nil {
		return err
	}

	sourceName, err := sourceFileName(cfg)
	if err != nil {
		return err
	}

	problemDir := filepath.Join(root, problem)
	samplesDir := filepath.Join(problemDir, "samples")
	if err := os.Mkdir(problemDir, 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(samplesDir, 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(problemDir, sourceName), template, 0o644); err != nil {
		return err
	}
	if err := createSampleFiles(samplesDir, 1, sampleCount); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "Created problem at %s\n", problemDir)
	return err
}

func cmdAddSamples(root, problem string, sampleCount int, stdout io.Writer) error {
	samplesDir := filepath.Join(root, problem, "samples")
	if _, err := os.Stat(samplesDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing samples directory: %s", samplesDir)
		}
		return err
	}

	start, err := nextSampleNumber(samplesDir)
	if err != nil {
		return err
	}
	if err := createSampleFiles(samplesDir, start, sampleCount); err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "Added %d sample(s) to %s\n", sampleCount, problem)
	return err
}
