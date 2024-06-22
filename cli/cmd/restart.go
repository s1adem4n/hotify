package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:               "restart",
	Short:             "Restart a service",
	Long:              `Restart a service, provide the name as the first argument.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: AutocompleteServiceName,
	Run: func(cmd *cobra.Command, args []string) {
		err := Client.RestartService(args[0])
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}
		fmt.Println("Service restarted")
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
