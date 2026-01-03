package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type HealthHandler struct {
	mongoClient *mongo.Client
	startTime   time.Time
	version     string
}

func NewHealthHandler(mongoClient *mongo.Client) *HealthHandler {
	version := os.Getenv("APP_VERSION")
	if version == "" {
		version = "unknown"
	}
	return &HealthHandler{
		mongoClient: mongoClient,
		startTime:   time.Now(),
		version:     version,
	}
}

// HealthResponse follows Kubernetes/OpenShift health check conventions
type HealthResponse struct {
	Status    string           `json:"status"`
	Timestamp string           `json:"timestamp"`
	Uptime    string           `json:"uptime"`
	Version   string           `json:"version"`
	Checks    map[string]Check `json:"checks"`
}

type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Health is for general health status (liveness probe in OpenShift)
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	checks := make(map[string]Check)
	status := "UP"
	httpStatus := http.StatusOK

	// Database check
	dbCheck := h.checkDatabase()
	checks["database"] = dbCheck
	if dbCheck.Status != "UP" {
		status = "DOWN"
		httpStatus = http.StatusServiceUnavailable
	}

	// Memory check
	memCheck := h.checkMemory()
	checks["memory"] = memCheck

	// Public key check
	keyCheck := h.checkPublicKey()
	checks["public_key"] = keyCheck
	if keyCheck.Status != "UP" {
		status = "DOWN"
		httpStatus = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    time.Since(h.startTime).Round(time.Second).String(),
		Version:   h.version,
		Checks:    checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// Ready checks if the service is ready to accept traffic (readiness probe in OpenShift)
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check MongoDB connection with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.mongoClient.Ping(ctx, nil); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "DOWN",
			"message": "Database not ready",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status": "UP",
	}); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// Live is a simple liveness check (liveness probe in OpenShift)
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "UP",
	})
}

func (h *HealthHandler) checkDatabase() Check {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.mongoClient.Ping(ctx, nil); err != nil {
		return Check{
			Status:  "DOWN",
			Message: "Cannot connect to database",
		}
	}
	return Check{Status: "UP"}
}

func (h *HealthHandler) checkMemory() Check {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := m.Alloc / 1024 / 1024
	return Check{
		Status:  "UP",
		Message: fmt.Sprintf("Allocated: %d MB", allocMB),
	}
}

func (h *HealthHandler) checkPublicKey() Check {
	keyPath := os.Getenv("PUBLIC_KEY_PATH")
	if keyPath == "" {
		keyPath = "/etc/certs/public.pem"
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return Check{
			Status:  "DOWN",
			Message: "Public key file not found",
		}
	}
	return Check{Status: "UP"}
}
