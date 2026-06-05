package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "ipdb-transform",
	Short: "ipdb-transform is a cli tool used for transforming ipdatabases from json format to mmdb format",
	Long:  "ipdb-transform is a cli tool used for transforming ipdatabases from json format to mmdb format. The source is in ipinfo format and the destination is in the format from maxmind.",
	Run:   func(cmd *cobra.Command, args []string) {},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal("An error occurred while executing ipdb-transform", err)
	}
}
