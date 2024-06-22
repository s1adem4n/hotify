package main

import (
	"flag"
	"hotify/pkg/api"
	"hotify/pkg/caddy"
	"hotify/pkg/config"
	s "hotify/pkg/services"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := flag.String("config", "config.toml", "Path to the config file")
	flag.Parse()

	var config config.Config
	err := config.Load(*configPath)
	if err != nil {
		slog.Error("Could not load config", "path", *configPath, "err", err)
		os.Exit(1)
	}

	caddyClient := caddy.NewClient(
		"srv0",
		"http://localhost:2019",
	)

	err = caddyClient.Init()
	if err != nil {
		slog.Error("Could not initialize Caddy client", "err", err)
	}

	manager := s.NewManager(&config, caddyClient)

	err = manager.Init()
	if err != nil {
		slog.Error("Could not initialize services", "err", err)
		os.Exit(1)
	}

	server := api.NewServer(&config, manager)

	go func() {
		err := server.Start()
		if err != nil {
			slog.Error("Could not start API server", "err", err)
			os.Exit(1)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-signals

	err = manager.Stop()
	if err != nil {
		slog.Error("Could not stop services", "err", err)
	}

	slog.Info("Exiting")
}
