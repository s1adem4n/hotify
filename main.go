package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Hotify is a service management tool for a single server.
// Features:
// - Reverse proxy management, using caddy
// - Service management
// - Webhook support

type Config struct {
	Services []ServiceConfig
	Address  string
}

func (c *Config) Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	err = toml.NewDecoder(file).Decode(c)

	return err
}

func CloneRepo(url string, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func PullRepo(dest string) error {
	cmd := exec.Command("git", "pull")
	cmd.Dir = dest
	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

type ServiceStatus int

const (
	ServiceStatusRunning ServiceStatus = iota
	ServiceStatusStopped
)

type Service struct {
	Config  ServiceConfig
	Process *os.Process
	Status  ServiceStatus
}

func (s *Service) Clone() error {
	slog.Info("Cloning service", "name", s.Config.Name)

	err := CloneRepo(s.Config.Repo, filepath.Join("services", s.Config.Name))
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) Pull() error {
	slog.Info("Pulling service", "name", s.Config.Name)

	err := PullRepo(filepath.Join("services", s.Config.Name))
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) Init() error {
	slog.Info("Initializing service", "name", s.Config.Name)

	root := filepath.Join("services", s.Config.Name)

	if _, err := os.Stat(root); os.IsNotExist(err) {
		err := os.MkdirAll(root, 0755)
		if err != nil {
			return err
		}

		err = s.Clone()
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) Update() error {
	slog.Info("Updating service", "name", s.Config.Name)

	err := s.Stop()
	if err != nil {
		return err
	}

	err = s.Pull()
	if err != nil {
		return err
	}

	err = s.Build()
	if err != nil {
		return err
	}

	err = s.Start()
	return err
}

func (s *Service) Build() error {
	slog.Info("Building service", "name", s.Config.Name)

	cmd := exec.Command("sh", "-c", s.Config.Build)
	cmd.Dir = filepath.Join("services", s.Config.Name)

	err := cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) Stop() error {
	slog.Info("Stopping service", "name", s.Config.Name)

	s.Status = ServiceStatusStopped

	// if process is running
	if s.Process != nil {
		waiting := make(chan bool)

		go func() {
			s.Process.Signal(syscall.SIGTERM)
			s.Process.Wait()
			close(waiting)
		}()

		select {
		case <-waiting:
		case <-time.After(5 * time.Second):
			slog.Info("Process did not exit after 5 seconds, killing", "name", s.Config.Name)
			s.Process.Kill()
		}
	}

	s.Process = nil

	return nil
}

func (s *Service) Start() error {
	slog.Info("Starting service", "name", s.Config.Name)

	cmd := exec.Command("sh", "-c", s.Config.Exec)
	cmd.Dir = filepath.Join("services", s.Config.Name)

	err := cmd.Start()
	if err != nil {
		return err
	}

	s.Process = cmd.Process
	go func() {
		state, _ := s.Process.Wait()

		if s.Status != ServiceStatusStopped {
			slog.Info("Service exited", "name", s.Config.Name, "code", state.ExitCode())

			if s.Config.Restart {
				s.Start()
			} else {
				s.Stop()
			}
		}
	}()

	return nil
}

type ServiceConfig struct {
	Name    string
	Repo    string
	Exec    string
	Build   string
	Restart bool
}

func main() {
	configPath := flag.String("config", "config.toml", "Path to the config file")
	flag.Parse()

	config := Config{}
	err := config.Load(*configPath)
	if err != nil {
		slog.Error("Could not load config", "path", *configPath, "err", err)
		os.Exit(1)
	}

	services := []*Service{}

	for _, service := range config.Services {
		s := Service{Config: service}
		err := s.Init()
		if err != nil {
			slog.Error("Could not initialize service", "name", s.Config.Name, "err", err)
			continue
		}

		err = s.Build()
		if err != nil {
			slog.Error("Could not build service", "name", s.Config.Name, "err", err)
			continue
		}

		err = s.Start()
		if err != nil {
			slog.Error("Could not start service", "name", s.Config.Name, "err", err)
			continue
		}

		services = append(services, &s)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/hooks/{service}", func(w http.ResponseWriter, r *http.Request) {
		service := r.PathValue("service")
		for _, s := range services {
			if s.Config.Name == service {
				go s.Update()

				w.WriteHeader(200)
				return
			}
		}

		w.WriteHeader(404)
	})

	go func() {
		err = http.ListenAndServe(config.Address, mux)
		if err != nil {
			slog.Error("Could not start server", "err", err)
			os.Exit(1)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	<-signals

	for _, s := range services {
		err := s.Stop()
		if err != nil {
			slog.Error("Could not stop service", "name", s.Config.Name, "err", err)
		}
	}

	slog.Info("Exiting")
}
