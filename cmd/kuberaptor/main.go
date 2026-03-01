package main

import (
	"fmt"
	"os"

	"github.com/magenx/kuberaptor/cmd/kuberaptor/commands"
	"github.com/magenx/kuberaptor/pkg/version"
)

func main() {
	if err := commands.Execute(version.Get()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
