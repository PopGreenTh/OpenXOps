package sense

import "github.com/spf13/cobra"

// Commands returns Sense-stage commands for xops.
func Commands() []*cobra.Command {
	sense := &cobra.Command{
		Use:   "sense",
		Short: "Read clouds and emit raw records",
	}
	sense.AddCommand(awsAccountsCommand())
	return []*cobra.Command{sense}
}
