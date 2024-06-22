package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:               "logs",
	Short:             "Get logs for a service",
	Long:              `Get logs for a service, provide the name as the first argument.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: AutocompleteServiceName,
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		live, _ := cmd.Flags().GetBool("live")

		if live {
			var numLogs int
			for {
				service, err := Client.Service(name)
				if err != nil {
					fmt.Printf("Error: %s\n", err)
					return
				}
				newLogs := service.Logs[numLogs:]
				for _, log := range newLogs {
					fmt.Print(log)
				}
				numLogs = len(service.Logs)

				time.Sleep(1 * time.Second)
			}
		}

		service, err := Client.Service(name)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}
		for _, log := range service.Logs {
			fmt.Print(log)
		}
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.Flags().BoolP("live", "l", false, "update logs in real-time")
}
