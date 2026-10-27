package main

import (
	"os"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/xops/sense"
)

func main() {
	root := record.Root("xops", "Cloud operations commands shared by every domain")
	root.AddCommand(sense.Commands()...)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
