package main

import (
	"fmt"
	"runtime"
	"strings"

	cli "github.com/urfave/cli/v3"
)

var (
	buildVersion   = "dev"
	buildCommit    = "unknown"
	buildGoVersion = ""
)

type versionInfo struct {
	Name      string `json:"name" yaml:"name"`
	Version   string `json:"version" yaml:"version"`
	Commit    string `json:"commit" yaml:"commit"`
	GoVersion string `json:"go_version" yaml:"go_version"`
}

func currentVersionInfo() versionInfo {
	version := strings.TrimSpace(buildVersion)
	if version == "" {
		version = "dev"
	}
	commit := strings.TrimSpace(buildCommit)
	if commit == "" {
		commit = "unknown"
	}
	goVersion := strings.TrimSpace(buildGoVersion)
	if goVersion == "" {
		goVersion = runtime.Version()
	}
	return versionInfo{
		Name:      appName,
		Version:   version,
		Commit:    commit,
		GoVersion: goVersion,
	}
}

func printVersion(_ *cli.Command) {
	info := currentVersionInfo()
	fmt.Printf("name: %s\n", info.Name)
	fmt.Printf("version: %s\n", info.Version)
	fmt.Printf("commit: %s\n", info.Commit)
	fmt.Printf("go: %s\n", info.GoVersion)
}
