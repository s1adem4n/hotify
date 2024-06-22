package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:               "start",
	Short:             "Start a service",
	Long:              `Start a service, provide the name as the first argument.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: AutocompleteServiceName,
	Run: func(cmd *cobra.Command, args []string) {
		err := Client.StartService(args[0])
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}
		fmt.Println("Service started")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
