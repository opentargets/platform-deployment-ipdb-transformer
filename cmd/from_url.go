package cmd

import "github.com/spf13/cobra"

var urlCmd = &cobra.Command{
	Use:     "url file",
	Aliases: []string{"u"},
	Short:   "Transform json file from url to mmdb",
	Long:    "Creates a Maxmind database with IP information using an url to directly fetch the database from IPInfo. The default output database name is 'output.mmdb'",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var output_path string
		if len(args) > 1 {
			output_path = args[1]
		} else {
			output_path = "output.mmdb"
		}
		fromUrl(args[0], output_path)
	},
}

func init() {
	rootCmd.AddCommand(urlCmd)
}
