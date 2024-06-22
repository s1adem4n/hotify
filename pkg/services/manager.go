package services

import (
	"hotify/pkg/caddy"
	"hotify/pkg/config"
	"path/filepath"
	"sync"
)

type Manager struct {
	Config   *config.Config
	Caddy    *caddy.Client
	services []*Service
	mu       sync.Mutex
}

func (m *Manager) InitService(service *Service) error {
	err := service.Init()
	if err != nil {
		return err
	}
	err = service.Build()
	if err != nil {
		return err
	}
	err = service.Start()
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) Init() error {
	for _, serviceConfig := range m.Config.Services {
		service := &Service{
			Config: &serviceConfig,
			Path:   filepath.Join(m.Config.ServicesPath, serviceConfig.Name),
			Caddy:  m.Caddy,
		}

		err := m.InitService(service)
		if err != nil {
			return err
		}

		m.mu.Lock()
		m.services = append(m.services, service)
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
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.services
}

func (m *Manager) Service(name string) *Service {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, service := range m.services {
		if service.Config.Name == name {
			return service
		}
	}

	return nil
}

func (m *Manager) Create(config *config.ServiceConfig) error {
	m.Config.Services = append(m.Config.Services, *config)
	err := m.Config.Save(m.Config.LoadPath)
	if err != nil {
		return err
	}

	service := &Service{
		Config: config,
		Path:   filepath.Join(m.Config.ServicesPath, config.Name),
	}

	err = m.InitService(service)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.services = append(m.services, service)
	m.mu.Unlock()

	return nil
}
