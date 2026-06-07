package main

import (
	"fmt"
	"io"
	"regexp"
	"runtime/debug"
)

var version = "dev"

var pseudoVersionPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\.\d{14}-[0-9a-f]+|-\d{14}-[0-9a-f]+)(?:\+dirty)?$`)

func isReleaseLikeVersion(value string) bool {
	if value == "" || value == "(devel)" {
		return false
	}
	if pseudoVersionPattern.MatchString(value) {
		return false
	}
	return true
}

func resolvedVersion() string {
	if version != "" && version != "dev" {
		return version
	}

	buildInfo, ok := debug.ReadBuildInfo()
	if ok && buildInfo != nil {
		if isReleaseLikeVersion(buildInfo.Main.Version) {
			return buildInfo.Main.Version
		}
	}

	return "dev"
}

func cmdVersion(stdout io.Writer) error {
	_, err := fmt.Fprintf(stdout, "cpx %s\n", resolvedVersion())
	return err
}
