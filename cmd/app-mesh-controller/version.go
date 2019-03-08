package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"runtime"
)

var (
	// These are set during build time via -ldflags
	version   string
	gitCommit string
	buildDate string
)

var (
	short      bool
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Version will output the current build information",
		Long:  ``,
		Run: func(_ *cobra.Command, _ []string) {
			version := NewVersion()
			if short {
				fmt.Printf("%+v", version.String())
			} else {
				fmt.Printf("%+v", version.JSON())
			}
		},
	}
)

func init() {
	versionCmd.Flags().BoolVarP(&short, "short", "s", false, "Print short output format for version information.")
	rootCmd.AddCommand(versionCmd)
}

type VersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	Compiler  string `json:"compiler"`
	Platform  string `json:"platform"`
}

func NewVersion() VersionInfo {
	return VersionInfo{
		Version:   version,
		GitCommit: gitCommit,
		BuildDate: buildDate,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func (v VersionInfo) String() string {
	return fmt.Sprintf("Version: %s\nGitCommit: %s\nBuildDate: %s\nGoVersion: %s\nCompiler: %s\nPlatform: %s\n",
		v.Version, v.GitCommit, v.BuildDate, v.GoVersion, v.Compiler, v.Platform)
}

func (v VersionInfo) JSON() string {
	marshalled, err := json.MarshalIndent(&v, "", "  ")
	if err != nil {
		fmt.Println("Error generating version JSON")
		os.Exit(1)
	}
	return string(marshalled)
}
