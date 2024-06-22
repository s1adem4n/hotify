package services

import (
	"bytes"
	"fmt"
	"hotify/pkg/caddy"
	"hotify/pkg/config"
	"hotify/pkg/git"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type ServiceStatus int

const (
	ServiceStatusRunning ServiceStatus = iota
	ServiceStatusStopped
)

type Service struct {
	Config   *config.ServiceConfig `json:"config"`
	Caddy    *caddy.Client         `json:"-"`
	Path     string                `json:"path"`
	Process  *os.Process           `json:"-"`
	Status   ServiceStatus         `json:"status"`
	Restarts int                   `json:"restarts"`
	Logs     []string              `json:"logs"`
}

func NewService(
	config *config.ServiceConfig,
	path string,
	caddy *caddy.Client,
) *Service {
	return &Service{
		Config: config,
		Caddy:  caddy,
		Path:   path,
		Logs:   []string{},
	}
}

func (s *Service) Clone() error {
	slog.Info("Cloning service", "name", s.Config.Name)

	err := git.CloneRepo(s.Config.Repo, s.Path)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) Pull() error {
	slog.Info("Pulling service", "name", s.Config.Name)

	err := git.PullRepo(s.Path)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) AddProxy() error {
	slog.Info("Adding service proxy", "name", s.Config.Name)

	if s.Config.Proxy.Match == "" {
		return nil
	}
	if s.Caddy.ObjectExists(fmt.Sprintf("id/%s", caddy.GenerateID(s.Config.Proxy.Match))) {
		return nil
	}

	proxy := caddy.NewProxy(
		caddy.GenerateID(s.Config.Proxy.Match),
		s.Config.Proxy.Match,
		s.Config.Proxy.Upstream,
	)

	err := s.Caddy.AddRoute(proxy)
	return err
}

func (s *Service) RemoveProxy() error {
	slog.Info("Removing service proxy", "name", s.Config.Name)

	if s.Config.Proxy.Match == "" {
		return nil
	}
	path := fmt.Sprintf("id/%s", caddy.GenerateID(s.Config.Proxy.Match))
	if !s.Caddy.ObjectExists(path) {
		return nil
	}

	err := s.Caddy.DeleteObject(path)
	return err
}

func (s *Service) Init() error {
	slog.Info("Initializing service", "name", s.Config.Name)

	if _, err := os.Stat(s.Path); os.IsNotExist(err) {
		err := os.MkdirAll(s.Path, 0755)
		if err != nil {
			return err
		}

		err = s.Clone()
		return err
	}

	err := s.Pull()
	return err
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

	err := s.RemoveProxy()
	if err != nil {
		return err
	}

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

type LogWriter struct {
	Service *Service
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	slog.Info(s)
	w.Service.Logs = append(w.Service.Logs, s)
	return len(p), nil
}

func (s *Service) Start() error {
	slog.Info("Starting service", "name", s.Config.Name)

	s.Status = ServiceStatusRunning

	err := s.AddProxy()
	if err != nil {
		return err
	}

	cmd := exec.Command("bash", "-c", s.Config.Exec)
	cmd.Dir = s.Path

	var writer LogWriter
	writer.Service = s
	cmd.Stdout = &writer
	cmd.Stderr = &writer

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("start failed: %s, err: %v", writer.Service.Logs, err)
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
