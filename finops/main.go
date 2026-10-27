package main

import (
	"os"

	"github.com/PopGreenTh/OpenXOps/record"
)

func main() {
	root := record.Root("finops", "FinOps commands for cloud cost")
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
