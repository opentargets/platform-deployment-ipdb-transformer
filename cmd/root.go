package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

// maxSkipRate is the maximum percentage of source records that may be skipped
// before the transformation fails. A negative value disables the check.
var maxSkipRate = 1.0

var rootCmd = &cobra.Command{
	Use:   "ipdb-transform",
	Short: "ipdb-transform is a cli tool used for transforming ipdatabases from json format to mmdb format",
	Long:  "ipdb-transform is a cli tool used for transforming ipdatabases from json format to mmdb format. The source is in ipinfo format and the destination is in the format from maxmind.",
	Run:   func(cmd *cobra.Command, args []string) {},
}

func init() {
	rootCmd.PersistentFlags().Float64Var(
		&maxSkipRate,
		"max-skip-rate",
		maxSkipRate,
		"maximum percentage of source records that may be skipped before the transformation fails (0 fails on any skipped record, a negative value disables the check)",
	)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal("An error occurred while executing ipdb-transform", err)
	}
}
