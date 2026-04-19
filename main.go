package main

import (
	"os"

	"github.com/zmorgan/umpire/cmd"
)

var Version = "dev"

func main() {
	cmd.SetVersion(Version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
