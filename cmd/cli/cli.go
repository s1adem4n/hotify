package main

// cli:
// services: list services
// services start <name>: start service
// services stop <name>: stop service
// services update <name>: update service

import (
	"flag"
	"fmt"
	"hotify/pkg/api"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Address string
	Secret  string
}

var configPath = flag.String("config", "config.toml", "Path to the config file")

type Table [][]string

func (t Table) Print() {
	for _, row := range t {
		for _, cell := range row {
			fmt.Printf("%-20s", cell)
		}
		fmt.Println()
	}
}

var statusMap = map[int]string{
	0: "running",
	1: "stopped",
}

func main() {
	flag.Parse()

	var config Config
	file, err := os.Open(*configPath)
	if err != nil {
		fmt.Println("Could not open config file", err)
		os.Exit(1)
	}
	defer file.Close()
	err = toml.NewDecoder(file).Decode(&config)
	if err != nil {
		fmt.Println("Could not decode config file", err)
		os.Exit(1)
	}

	client := api.NewClient("http://localhost:1234", config.Secret)

	// handle services command
	if len(os.Args) == 0 {
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
	default:
		fmt.Println("Usage: hotify <command> [args]")
		os.Exit(1)
	}
}
