package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var fileCmd = &cobra.Command{
	Use:     "file",
	Aliases: []string{"f"},
	Short:   "Transform json file from local path to mmdb",
	Long:    "Creates a Maxmind database with IP information using a file containing an IPInfo database.",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var output_path string
		if len(args) > 1 {
			output_path = args[1]
		} else {
			output_path = "output.mmdb"
		}
		if _, err := fromFile(args[0], output_path); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(fileCmd)
}
