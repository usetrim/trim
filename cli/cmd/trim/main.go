package main

import (
	"fmt"
	"os"

	"github.com/usetrim/trim/cli/internal/cli"
	"github.com/usetrim/trim/cli/internal/clierr"
)

func main() {
	if err := cli.Execute(); err != nil {
		if ex, ok := clierr.AsExit(err); ok {
			// Commands print human messages themselves; honor the exit code only.
			_ = ex
			os.Exit(ex.Code)
		}
		if fmtStr := cli.MainErrFmt(); fmtStr != "" {
			fmt.Fprintf(os.Stderr, fmtStr, err)
		} else {
			// Fail-closed: no invent "trim: " prefix when chrome unavailable.
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		os.Exit(1)
	}
}
