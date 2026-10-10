package main

import (
	"fmt"
	"os"

	"github.com/usetrim/trim/cli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		if fmtStr := cli.MainErrFmt(); fmtStr != "" {
			fmt.Fprintf(os.Stderr, fmtStr, err)
		} else {
			// Fail-closed: no invent "trim: " prefix when chrome unavailable.
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		os.Exit(1)
	}
}
