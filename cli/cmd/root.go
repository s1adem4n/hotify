package cmd

import (
	"hotify/cli/config"
	"hotify/pkg/api"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var Client *api.Client

var rootCmd = &cobra.Command{
	Use:   "hotify",
	Short: "Manage your hotify instance",
	Long:  `Use this CLI to manage your hotify instance. You can start, stop, and update services, as well as view logs and other information.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	path := filepath.Join(configDir, "hotify", "config.toml")

	var config config.Config
	config.Load(path)
	Client = api.NewClient(config.Address, config.Secret)

	rootCmd.PersistentFlags().StringP("config", "c", path, "Path to the config file")

	if config.Address == "" || config.Secret == "" {
		PrintlnBold("You must configure the CLI before using it.\n")
		configureCmd.Run(nil, nil)
	}
}
