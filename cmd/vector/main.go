package main

import (
	"os"

	"github.com/built-fast/vector-cli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
