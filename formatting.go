package main

import (
	"fmt"
	"io"
)

type commandHelpEntry struct {
	command     string
	description string
}

func maxWidth(values []string) int {
	width := 0
	for _, value := range values {
		if len(value) > width {
			width = len(value)
		}
	}
	return width
}

func writeAlignedCommandTable(w io.Writer, entries []commandHelpEntry) error {
	commands := make([]string, 0, len(entries))
	for _, entry := range entries {
		commands = append(commands, entry.command)
	}
	width := maxWidth(commands)
	for _, entry := range entries {
		if _, err := fmt.Fprintf(w, "  %-*s  %s\n", width, entry.command, entry.description); err != nil {
			return err
		}
	}
	return nil
}

func formatAlignedField(label, value string, width int) string {
	return fmt.Sprintf("  %-*s %s", width, label+":", value)
}
