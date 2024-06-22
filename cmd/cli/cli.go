package main

// cli:
// services: list services
// services start <name>: start service
// services stop <name>: stop service
// services update <name>: update service

import (
	"bufio"
	"flag"
	"fmt"
	"hotify/pkg/api"
	"hotify/pkg/config"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Address string
	Secret  string
}

var configPath = flag.String("config", "", "Path to the config file")

type Table [][]string

func (t Table) Print() {
	for i, row := range t {
		for _, cell := range row {
			if i == 0 {
				fmt.Printf("\033[1m%-20s\033[0m", cell)
			} else {
				fmt.Printf("%-20s", cell)
			}
		}
		fmt.Println()
	}
}

var statusMap = map[int]string{
	0: "running",
	1: "stopped",
}

func PrintlnBold(text string) {
	fmt.Printf("\033[1m%s\033[0m\n", text)
}

func InputPrompt(prompt string, dest *string) {
	PrintlnBold(prompt)
	var s string
	r := bufio.NewReader(os.Stdin)
	for {
		s, _ = r.ReadString('\n')
		if s != "" {
			break
		}
	}
	*dest = strings.TrimSpace(s)
}

func main() {
	flag.Parse()

	if *configPath == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			fmt.Println("Could not get user config dir", err)
			os.Exit(1)
		}

		path := filepath.Join(configDir, "hotify", "config.toml")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			var conf Config

			os.MkdirAll(filepath.Dir(path), 0755)

			file, err := os.Create(path)
			if err != nil {
				fmt.Println("Could not create config file", err)
				os.Exit(1)
			}
			defer file.Close()

			InputPrompt("Address", &conf.Address)
			InputPrompt("Secret", &conf.Secret)

			err = toml.NewEncoder(file).Encode(conf)
			if err != nil {
				fmt.Println("Could not encode config file", err)
				os.Exit(1)
			}
		}

		*configPath = path
	}

	var conf Config
	file, err := os.Open(*configPath)
	if err != nil {
		fmt.Println("Could not open config file", err)
		os.Exit(1)
	}
	defer file.Close()
	err = toml.NewDecoder(file).Decode(&conf)
	if err != nil {
		fmt.Println("Could not decode config file", err)
		os.Exit(1)
	}

	client := api.NewClient("http://localhost:1234", "secret")

	// handle services command
	if len(os.Args) <= 1 {
		fmt.Println("Usage: hotify <command> [args]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		services, err := client.Services()
		if err != nil {
			fmt.Println("Could not list services", err)
			os.Exit(1)
		}
		table := Table{}

		table = append(table, []string{"Name", "Status", "Restarts"})
		for _, service := range services {
			table = append(table, []string{service.Config.Name, statusMap[int(service.Status)], fmt.Sprintf("%d", service.Restarts)})
		}
		table.Print()
	case "start":
		if len(os.Args) < 3 {
			fmt.Println("Usage: hotify start <name>")
			os.Exit(1)
		}
		err := client.StartService(os.Args[2])
		if err != nil {
			fmt.Println("Could not start service", err)
			os.Exit(1)
		}
		fmt.Println("Service started")
	case "stop":
		if len(os.Args) < 3 {
			fmt.Println("Usage: hotify stop <name>")
			os.Exit(1)
		}
		err := client.StopService(os.Args[2])
		if err != nil {
			fmt.Println("Could not stop service", err)
			os.Exit(1)
		}
		fmt.Println("Service stopped")
	case "update":
		if len(os.Args) < 3 {
			fmt.Println("Usage: hotify update <name>")
			os.Exit(1)
		}
		err := client.UpdateService(os.Args[2])
		if err != nil {
			fmt.Println("Could not update service", err)
			os.Exit(1)
		}
		fmt.Println("Service updated")
	case "logs":
		if len(os.Args) < 3 {
			fmt.Println("Usage: hotify logs <name>")
			os.Exit(1)
		}
		service, err := client.Service(os.Args[2])
		if err != nil {
			fmt.Println("Could not get service", err)
			os.Exit(1)
		}
		for _, log := range service.Logs {
			fmt.Print(log)
		}
		os.Exit(0)
	case "create":
		var config config.ServiceConfig
		// prevent the shell from trying to interpret the input as a command
		fmt.Print("\033[1m")
		defer fmt.Print("\033[0m")

		InputPrompt("Name", &config.Name)
		InputPrompt("Repo", &config.Repo)
		InputPrompt("Build command", &config.Build)
		InputPrompt("Exec command", &config.Exec)

		var proxy string
		InputPrompt("Proxy (y/n)", &proxy)
		if proxy == "y" {
			InputPrompt("Proxy match", &config.Proxy.Match)
			InputPrompt("Proxy upstream", &config.Proxy.Upstream)
		}

		err := client.CreateService(&config)
		if err != nil {
			fmt.Println("Could not create service", err)
			os.Exit(1)
		}
		fmt.Println("Service created")
		os.Exit(0)
	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Usage: hotify delete <name>")
			os.Exit(1)
		}
		err := client.DeleteService(os.Args[2])
		if err != nil {
			fmt.Println("Could not delete service", err)
			os.Exit(1)
		}
		fmt.Println("Service deleted")
		os.Exit(0)
	case "restart":
		if len(os.Args) < 3 {
			fmt.Println("Usage: hotify restart <name>")
			os.Exit(1)
		}
		err := client.RestartService(os.Args[2])
		if err != nil {
			fmt.Println("Could not restart service", err)
			os.Exit(1)
		}
		fmt.Println("Service restarted")
		os.Exit(0)
	default:
		fmt.Println("Usage: hotify <command> [args]")
		os.Exit(1)
	}
}
