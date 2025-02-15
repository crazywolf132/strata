package main

import (
	"fmt"
	"os"
	"strata/cmd"
	"strata/internal/logs"
)

func main() {
	if err := cmd.Execute(); err != nil {
		logs.Error("CLI error: %v", err)
		fmt.Fprintln(os.Stderr, "Error: ", err)
		os.Exit(1)
	}
}
