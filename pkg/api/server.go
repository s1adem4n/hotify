package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hotify/pkg/config"
	"hotify/pkg/services"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

func RespondJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func VerifyRequest(body []byte, signatureHeader string, secret string) bool {
	signature := hmac.New(sha256.New, []byte(secret))
	signature.Write([]byte(body))

	expected := fmt.Sprintf("sha256=%x", signature.Sum(nil))
	return signatureHeader == expected
}

type Server struct {
	Config  *config.Config
	Manager *services.Manager
	mux     *http.ServeMux
}

func NewServer(config *config.Config, manager *services.Manager) *Server {
	s := &Server{
		Config:  config,
		Manager: manager,
		mux:     http.NewServeMux(),
	}

	s.mux.HandleFunc("GET /api/config", s.GetConfig)

	s.mux.HandleFunc("GET /api/services", s.GetServices)
	s.mux.HandleFunc("POST /api/services", s.CreateService)

	s.mux.HandleFunc("GET /api/services/{service}", s.GetService)
	s.mux.HandleFunc("DELETE /api/services/{service}", s.DeleteService)

	s.mux.HandleFunc("GET /api/services/{service}/start", s.StartService)
	s.mux.HandleFunc("GET /api/services/{service}/stop", s.StopService)
	s.mux.HandleFunc("GET /api/services/{service}/update", s.UpdateService)
	s.mux.HandleFunc("GET /api/services/{service}/restart", s.RestartService)

	s.mux.HandleFunc("/hooks/{service}", s.ServiceWebhook)

	return s
}

func (s *Server) Start() error {
	slog.Info("Starting API server", "address", s.Config.Address)
	err := http.ListenAndServe(s.Config.Address, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Signature-256")

		slog.Info("Request", "method", r.Method, "path", r.URL.Path)

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/api") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				RespondJSON(w, http.StatusInternalServerError, nil)
				return
			}

			if !VerifyRequest(body, r.Header.Get("X-Signature-256"), s.Config.Secret) {
				RespondJSON(w, http.StatusForbidden, nil)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
		}

		s.mux.ServeHTTP(w, r)
	}))

	return err
}

func (s *Server) GetConfig(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, http.StatusOK, s.Config)
}

func (s *Server) GetServices(w http.ResponseWriter, r *http.Request) {
	services := s.Manager.Services()

	RespondJSON(w, http.StatusOK, services)
}

func (s *Server) GetService(w http.ResponseWriter, r *http.Request) {
	service := s.Manager.Service(r.PathValue("service"))

	if service == nil {
		RespondJSON(w, http.StatusNotFound, nil)
		return
	}

	RespondJSON(w, http.StatusOK, service)
}

func (s *Server) CreateService(w http.ResponseWriter, r *http.Request) {
	var serviceConfig config.ServiceConfig
	err := json.NewDecoder(r.Body).Decode(&serviceConfig)
	if err != nil {
		RespondJSON(w, http.StatusBadRequest, nil)
		return
	}

	if s.Manager.Service(serviceConfig.Name) != nil {
		RespondJSON(w, http.StatusConflict, nil)
		return
	}

	err = s.Manager.Create(&serviceConfig)
	if err != nil {
		slog.Error("Failed to create service", "error", err)
		RespondJSON(w, http.StatusInternalServerError, nil)
		return
	}

	RespondJSON(w, http.StatusCreated, nil)
}

func (s *Server) StartService(w http.ResponseWriter, r *http.Request) {
	service := s.Manager.Service(r.PathValue("service"))
	if service == nil {
		RespondJSON(w, http.StatusNotFound, nil)
		return
	}

	err := service.Start()
	if err != nil {
		slog.Error("Failed to start service", "error", err)
		RespondJSON(w, http.StatusInternalServerError, nil)
		return
	}

	RespondJSON(w, http.StatusOK, nil)
}

func (s *Server) StopService(w http.ResponseWriter, r *http.Request) {
	service := s.Manager.Service(r.PathValue("service"))
	if service == nil {
		RespondJSON(w, http.StatusNotFound, nil)
		return
	}

	err := service.Stop()
	if err != nil {
		slog.Error("Failed to stop service", "error", err)
		RespondJSON(w, http.StatusInternalServerError, nil)
		return
	}

	RespondJSON(w, http.StatusOK, nil)
}

func (s *Server) UpdateService(w http.ResponseWriter, r *http.Request) {
	service := s.Manager.Service(r.PathValue("service"))
	if service == nil {
		RespondJSON(w, http.StatusNotFound, nil)
		return
	}

	err := service.Update()
	if err != nil {
		slog.Error("Failed to update service", "error", err)
		RespondJSON(w, http.StatusInternalServerError, nil)
		return
	}

	RespondJSON(w, http.StatusOK, nil)
}

func (s *Server) DeleteService(w http.ResponseWriter, r *http.Request) {
	service := s.Manager.Service(r.PathValue("service"))
	if service == nil {
		RespondJSON(w, http.StatusNotFound, nil)
		return
	}

	err := s.Manager.Delete(r.PathValue("service"))
	if err != nil {
		slog.Error("Failed to delete service", "error", err)
		RespondJSON(w, http.StatusInternalServerError, nil)
		return
	}

	RespondJSON(w, http.StatusOK, nil)
}

func (s *Server) RestartService(w http.ResponseWriter, r *http.Request) {
	service := s.Manager.Service(r.PathValue("service"))
	if service == nil {
		RespondJSON(w, http.StatusNotFound, nil)
		return
	}

	err := service.Restart()
	if err != nil {
		slog.Error("Failed to restart service", "error", err)
		RespondJSON(w, http.StatusInternalServerError, nil)
		return
	}

	RespondJSON(w, http.StatusOK, nil)
}

func (s *Server) ServiceWebhook(w http.ResponseWriter, r *http.Request) {
	signatureHeader := r.Header.Get("X-Hub-Signature-256")

	name := r.PathValue("service")
	service := s.Manager.Service(name)
	if service == nil {
		w.WriteHeader(404)
		return
	}

	if service.Config.Secret != "" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		if !VerifyRequest(body, signatureHeader, service.Config.Secret) {
			slog.Warn("Invalid signature", "service", service.Config.Name)
			w.WriteHeader(403)
			return
		}
	}

	slog.Info("Received webhook", "service", service.Config.Name)
	err := service.Update()
	if err != nil {
		slog.Error("Failed to update service", "error", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
}
