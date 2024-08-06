package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"hotify/pkg/config"
	"hotify/pkg/services"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func VerifyRequest(body []byte, signatureHeader string, secret string) bool {
	signature := hmac.New(sha256.New, []byte(secret))
	signature.Write([]byte(body))

	expected := fmt.Sprintf("sha256=%x", signature.Sum(nil))
	return signatureHeader == expected
}

type Server struct {
	Config  *config.Config
	Manager *services.Manager
	Group   *echo.Group
}

func NewServer(config *config.Config, manager *services.Manager, group *echo.Group) *Server {
	s := &Server{
		Config:  config,
		Manager: manager,
		Group:   group,
	}

	// auth middleware
	s.Group.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// ignore webhooks
			if strings.HasSuffix(c.Request().URL.Path, "/webhook") {
				return next(c)
			}

			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				slog.Error("Failed to read body", "error", err)
				return c.JSON(http.StatusInternalServerError, nil)
			}

			if !VerifyRequest(body, c.Request().Header.Get("X-Signature-256"), s.Config.Secret) {
				return c.JSON(http.StatusForbidden, nil)
			}

			c.Request().Body = io.NopCloser(bytes.NewReader(body))

			return next(c)
		}
	})

	s.Group.GET("/config", s.GetConfig)

	s.Group.GET("/services", s.GetServices)
	s.Group.POST("/services", s.CreateService)

	s.Group.GET("/services/:service", s.GetService)
	s.Group.DELETE("/services/:service", s.DeleteService)

	s.Group.GET("/services/:service/start", s.StartService)
	s.Group.GET("/services/:service/stop", s.StopService)
	s.Group.GET("/services/:service/update", s.UpdateService)
	s.Group.GET("/services/:service/restart", s.RestartService)

	s.Group.POST("/services/:service/webhook", s.ServiceWebhook)

	return s
}

func (s *Server) GetConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, s.Config)
}

func (s *Server) GetServices(c echo.Context) error {
	services := s.Manager.Services()

	return c.JSON(http.StatusOK, services)
}

func (s *Server) GetService(c echo.Context) error {
	service := s.Manager.Service(
		c.Param("service"),
	)

	if service == nil {
		return c.JSON(http.StatusNotFound, nil)
	}

	return c.JSON(http.StatusOK, service)
}

func (s *Server) CreateService(c echo.Context) error {
	var serviceConfig config.ServiceConfig
	if err := c.Bind(&serviceConfig); err != nil {
		return c.JSON(http.StatusBadRequest, nil)
	}

	if s.Manager.Service(serviceConfig.Name) != nil {
		return c.JSON(http.StatusConflict, nil)
	}

	err := s.Manager.Create(&serviceConfig)
	if err != nil {
		slog.Error("Failed to create service", "error", err)
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, nil)
}

func (s *Server) StartService(c echo.Context) error {
	service := s.Manager.Service(c.Param("service"))
	if service == nil {
		return c.JSON(http.StatusNotFound, nil)
	}

	err := service.Start()
	if err != nil {
		slog.Error("Failed to start service", "error", err)
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, nil)
}

func (s *Server) StopService(c echo.Context) error {
	service := s.Manager.Service(c.Param("service"))
	if service == nil {
		return c.JSON(http.StatusNotFound, nil)
	}

	err := service.Stop()
	if err != nil {
		slog.Error("Failed to stop service", "error", err)
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, nil)
}

func (s *Server) UpdateService(c echo.Context) error {
	service := s.Manager.Service(c.Param("service"))
	if service == nil {
		return c.JSON(http.StatusNotFound, nil)
	}

	err := service.Update()
	if err != nil {
		slog.Error("Failed to update service", "error", err)
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, nil)
}

func (s *Server) DeleteService(c echo.Context) error {
	service := s.Manager.Service(c.Param("service"))
	if service == nil {
		return c.JSON(http.StatusNotFound, nil)
	}

	err := s.Manager.Delete(service.Config.Name)
	if err != nil {
		slog.Error("Failed to delete service", "error", err)
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, nil)
}

func (s *Server) RestartService(c echo.Context) error {
	service := s.Manager.Service(c.Param("service"))
	if service == nil {
		return c.JSON(http.StatusNotFound, nil)
	}

	err := service.Restart()
	if err != nil {
		slog.Error("Failed to restart service", "error", err)
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, nil)
}

func (s *Server) ServiceWebhook(c echo.Context) error {
	signatureHeader := c.Request().Header.Get("X-Hub-Signature-256")

	name := c.Param("service")
	service := s.Manager.Service(name)
	if service == nil {
		return c.JSON(http.StatusNotFound, nil)
	}

	if service.Config.Secret != "" {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			slog.Error("Failed to read body", "error", err)
			return c.JSON(http.StatusInternalServerError, nil)
		}

		if !VerifyRequest(body, signatureHeader, service.Config.Secret) {
			slog.Warn("Invalid signature", "service", service.Config.Name, "signature", signatureHeader, "expected", service.Config.Secret)
			return c.JSON(http.StatusForbidden, nil)
		}
	}

	slog.Info("Received webhook", "service", service.Config.Name)
	err := service.Update()
	if err != nil {
		slog.Error("Failed to update service", "error", err)
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, nil)
}
