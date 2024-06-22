package cmd

import (
	"fmt"
	"hotify/pkg/config"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a service",
	Long:  `Create a service interactively.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var config config.ServiceConfig
		Prompt("Service name", &config.Name)
		Prompt("Repository", &config.Repo)
		Prompt("Exec command", &config.Exec)
		Prompt("Build command", &config.Build)
		Prompt("Webhook secret", &config.Secret)

		PromptBool("Restart on failure", &config.Restart)
		if config.Restart {
			PromptInt("Max restarts", &config.MaxRestarts)
		}

		var proxy bool
		PromptBool("Use proxy", &proxy)
		if proxy {
			Prompt("Match", &config.Proxy.Match)
			Prompt("Upstream", &config.Proxy.Upstream)
		}

		err := Client.CreateService(&config)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return
		}
		fmt.Println("Service created")
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
