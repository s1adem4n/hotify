package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:               "delete",
	Short:             "Delete a service",
	Long:              `Delete a service, provide the name as the first argument.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: AutocompleteServiceName,
	Run: func(cmd *cobra.Command, args []string) {
		err := Client.DeleteService(args[0])
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}
		fmt.Println("Service deleted")
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
