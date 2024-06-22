package cmd

import (
	"fmt"
	"hotify/cli/config"
	"os"

	"github.com/spf13/cobra"
)

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure the hotify CLI",
	Long:  `Enter your server address and API secret to use the hotify CLI.`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		configPath, _ := cmd.Flags().GetString("config")
		if _, err := os.Stat(configPath); err == nil {
			os.Remove(configPath)
		}

		file, err := os.Create(configPath)
		if err != nil {
			fmt.Println("Could not create config file")
			return
		}
		defer file.Close()

		var config config.Config
		Prompt("Server address", &config.Address)
		Prompt("API secret", &config.Secret)
		config.Save(configPath)

		fmt.Println("\nConfiguration saved")
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)
}
