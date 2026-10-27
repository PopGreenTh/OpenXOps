// Command doccheck verifies user docs against code (WI-0108, D-044).
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := Run("."); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
