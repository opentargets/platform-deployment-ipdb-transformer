package cmd

import "github.com/spf13/cobra"

var fileCmd = &cobra.Command{
	Use:     "file",
	Aliases: []string{"f"},
	Short:   "Transform json file from local path to mmdb",
	Long:    "pending..",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var output_path string
		if len(args) > 1 {
			output_path = args[1]
		} else {
			output_path = "output.mmdb"
		}
		fromFile(args[0], output_path)
	},
}

func init() {
	rootCmd.AddCommand(fileCmd)
}
