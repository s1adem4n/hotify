package main

import (
	"flag"
	"hotify/pkg/api"
	"hotify/pkg/caddy"
	"hotify/pkg/config"
	s "hotify/pkg/services"
	"hotify/webui"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func SPAMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := next(c)
		if err != nil {
			if he, ok := err.(*echo.HTTPError); ok {
				if he.Code == http.StatusNotFound {
					c.Request().URL.Path = "/"
					return next(c)
				}
			}
		}
		return err
	}
}

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

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "X-Signature-256"},
	}))
	e.Pre(SPAMiddleware)

	apiGroup := e.Group("/api")

	api.NewServer(&config, manager, apiGroup)

	frontend := echo.MustSubFS(webui.Assets, "build")
	e.StaticFS("/", frontend)

	go func() {
		slog.Info("Starting server", "address", config.Address)
		err := e.Start(config.Address)
		if err != nil {
			slog.Error("Could not start server", "err", err)
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
