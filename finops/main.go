package main

import (
	"os"

	"github.com/PopGreenTh/OpenXOps/finops/enrich"
	"github.com/PopGreenTh/OpenXOps/finops/sense"
	"github.com/PopGreenTh/OpenXOps/record"
)

func main() {
	root := record.Root("finops", "FinOps commands for cloud cost")
	root.AddCommand(sense.Commands()...)
	root.AddCommand(enrich.Commands()...)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
