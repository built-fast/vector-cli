package main

import (
	"os"

	"github.com/built-fast/vector-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
