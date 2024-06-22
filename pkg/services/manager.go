package services

import (
	"errors"
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

func NewManager(config *config.Config, caddy *caddy.Client) *Manager {
	return &Manager{
		Config: config,
		Caddy:  caddy,
	}
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
		service := NewService(
			&serviceConfig,
			filepath.Join(m.Config.ServicesPath, serviceConfig.Name),
			m.Caddy,
		)
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

	service := NewService(
		config,
		filepath.Join(m.Config.ServicesPath, config.Name),
		m.Caddy,
	)

	err = m.InitService(service)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.services = append(m.services, service)
	m.mu.Unlock()

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

	for i, s := range m.services {
		if s.Config.Name == name {
			m.services = append(m.services[:i], m.services[i+1:]...)
			break
		}
	}

	m.Config.Services = []config.ServiceConfig{}
	for _, s := range m.services {
		m.Config.Services = append(m.Config.Services, *s.Config)
	}

	err = m.Config.Save(m.Config.LoadPath)
	if err != nil {
		return err
	}

	return nil
}
