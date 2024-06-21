package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
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

type Config struct {
	Services     []ServiceConfig
	Address      string
	ServicesPath string
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
	Config   ServiceConfig
	Path     string
	Process  *os.Process
	Status   ServiceStatus
	Restarts int
}

func (s *Service) Clone() error {
	slog.Info("Cloning service", "name", s.Config.Name)

	err := CloneRepo(s.Config.Repo, s.Path)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) Pull() error {
	slog.Info("Pulling service", "name", s.Config.Name)

	err := PullRepo(s.Path)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) Init() error {
	slog.Info("Initializing service", "name", s.Config.Name)

	if _, err := os.Stat(s.Path); os.IsNotExist(err) {
		err := os.MkdirAll(s.Path, 0755)
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

	cmd := exec.Command("bash", "-c", s.Config.Build)
	cmd.Dir = s.Path

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("build failed: %s, err: %v", buf.String(), err)
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

	s.Status = ServiceStatusRunning

	cmd := exec.Command("bash", "-c", s.Config.Exec)
	cmd.Dir = s.Path

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("start failed: %s, err: %v", buf.String(), err)
	}

	s.Process = cmd.Process
	go func() {
		state, _ := s.Process.Wait()

		if s.Status == ServiceStatusStopped {
			return
		}

		slog.Info("Service exited", "name", s.Config.Name, "code", state.ExitCode())

		if !s.Config.Restart {
			s.Stop() // mark as stopped, free resources
			return
		}

		if s.Restarts >= s.Config.MaxRestarts {
			slog.Error("Service reached max restarts", "name", s.Config.Name)
			s.Stop()
			return
		}

		s.Restarts++
		slog.Info("Restarting service", "name", s.Config.Name, "restarts", s.Restarts)
		s.Start()
	}()

	return nil
}

type ServiceConfig struct {
	// Name of the service, used for logging and folder name
	Name string
	// Git repository URL
	Repo string
	// Command to execute to build the service, relative to the git repository
	Exec string
	// Command to execute to build the service, relative to the git repository
	Build string
	// Restart the service when it exits
	Restart bool
	// Maximum number of restarts before giving up
	MaxRestarts int
	// Webhook secret to trigger updates
	Secret string
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
		s := Service{Config: service, Path: filepath.Join(config.ServicesPath, service.Name)}

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
		signatureHeader := r.Header.Get("X-Hub-Signature-256")

		service := r.PathValue("service")
		for _, s := range services {
			if s.Config.Name == service {
				if s.Config.Secret != "" {
					body, err := io.ReadAll(r.Body)
					if err != nil {
						w.WriteHeader(500)
						return
					}

					signature := hmac.New(sha256.New, []byte(s.Config.Secret))
					signature.Write([]byte(body))

					expected := fmt.Sprintf("sha256=%x", signature.Sum(nil))
					if signatureHeader != expected {
						slog.Warn("Invalid signature", "service", s.Config.Name)
						w.WriteHeader(403)
						return
					}
				}

				slog.Info("Received webhook", "service", s.Config.Name)
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
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-signals

	for _, s := range services {
		err := s.Stop()
		if err != nil {
			slog.Error("Could not stop service", "name", s.Config.Name, "err", err)
		}
	}

	slog.Info("Exiting")
}
