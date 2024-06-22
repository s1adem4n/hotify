package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:               "stop",
	Short:             "Stop a service",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: AutocompleteServiceName,
	Long:              `Stop a service, provide the name as the first argument.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := Client.StopService(args[0])
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}
		fmt.Println("Service stopped")
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
