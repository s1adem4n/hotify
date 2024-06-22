package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statusMap = map[int]string{
	0: "running",
	1: "stopped",
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Display all your services",
	Long:  `Display all your services in a table`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		services, err := Client.Services()
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}
		var table Table
		table = append(table, []string{"Name", "Status", "Restarts"})
		for _, service := range services {
			table = append(
				table,
				[]string{
					service.Config.Name,
					statusMap[int(service.Status)],
					fmt.Sprintf("%d", service.Restarts),
				},
			)
		}
		table.Print()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
