package main

import (
	"os"

	"github.com/PopGreenTh/OpenXOps/record"
)

func main() {
	root := record.Root("xops", "Cloud operations commands shared by every domain")
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
