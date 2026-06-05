package main

import (
	"fmt"
	"io"
)

var version = "dev"

func cmdVersion(stdout io.Writer) error {
	_, err := fmt.Fprintf(stdout, "cpx %s\n", version)
	return err
}
