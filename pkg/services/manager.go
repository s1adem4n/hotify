package services

import (
	"errors"
	"hotify/pkg/caddy"
	"hotify/pkg/config"
	"hotify/pkg/git"
	"log/slog"
	"path/filepath"
	"sync"
)

type Manager struct {
	Config   *config.Config
	Caddy    *caddy.Client
	services map[string]*Service
	mu       sync.Mutex
}

func NewManager(config *config.Config, caddy *caddy.Client) *Manager {
	return &Manager{
		Config:   config,
		Caddy:    caddy,
		services: make(map[string]*Service),
	}
}

func (m *Manager) InitService(service *Service) error {
	err := service.Init()
	if err != nil {
		return err
	}

	isNewestCommit, err := git.IsNewestCommit(service.Path)
	if err != nil {
		return err
	}
	if !isNewestCommit || service.Config.InitialBuild {
		err = service.Pull()
		if err != nil {
			return err
		}

		err = service.Build()
		if err != nil {
			return err
		}

		service.Config.InitialBuild = false
		m.Config.Save(m.Config.LoadPath)
	} else {
		slog.Info("Service is up to date, skipping build", "name", service.Config.Name)
	}

	err = service.Start()
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) Init() error {
	for key, serviceConfig := range m.Config.Services {
		service := NewService(
			serviceConfig,
			filepath.Join(m.Config.ServicesPath, serviceConfig.Name),
			m.Caddy,
		)
		err := m.InitService(service)
		if err != nil {
			return err
		}

		m.mu.Lock()
		m.services[key] = service
		m.mu.Unlock()
	}

	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, service := range m.services {
		err := service.Stop()
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) Services() []*Service {
	var services []*Service
	for _, service := range m.services {
		services = append(services, service)
	}

	return services
}

func (m *Manager) Service(name string) *Service {
	return m.services[name]
}

func (m *Manager) Create(config *config.ServiceConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Service(config.Name) != nil {
		return errors.New("service already exists")
	}

	m.Config.Services[config.Name] = config
	err := m.Config.Save(m.Config.LoadPath)
	if err != nil {
		return err
	}

	service := NewService(
		config,
		filepath.Join(m.Config.ServicesPath, config.Name),
		m.Caddy,
	)

	err = m.InitService(service)
	if err != nil {
		return err
	}

	m.services[config.Name] = service

	return nil
}

func (m *Manager) Delete(name string) error {
	service := m.Service(name)
	if service == nil {
		return errors.New("service not found")
	}

	err := service.Remove()
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.services, name)
	delete(m.Config.Services, name)

	err = m.Config.Save(m.Config.LoadPath)
	if err != nil {
		return err
	}

	return nil
}
