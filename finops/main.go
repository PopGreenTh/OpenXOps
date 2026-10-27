package main

import (
	"os"

	"github.com/PopGreenTh/OpenXOps/finops/analyze"
	"github.com/PopGreenTh/OpenXOps/finops/enrich"
	"github.com/PopGreenTh/OpenXOps/record"
)

func main() {
	root := record.Root("finops", "FinOps commands for cloud cost")
	root.AddCommand(enrich.Commands()...)
	root.AddCommand(analyze.Commands()...)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
