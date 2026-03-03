package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handlers holds handler dependencies.
type Handlers struct {
	store Store
}

// ---- Models ----

// Device represents an enrolled endpoint.
type Device struct {
	ID         string    `json:"id"`
	Hostname   string    `json:"hostname"`
	OS         string    `json:"os"`
	IP         string    `json:"ip"`
	AgentToken string    `json:"-"`
	EnrolledAt time.Time `json:"enrolled_at"`
	LastSeen   time.Time `json:"last_seen"`
}

// Command represents a server-issued command queued for a device.
type Command struct {
	ID        string          `json:"id"`
	DeviceID  string          `json:"device_id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Status    string          `json:"status"`
	Result    json.RawMessage `json:"result,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ---- Request/Response types ----

type enrollRequest struct {
	Hostname  string `json:"hostname"`
	OS        string `json:"os"`
	IP        string `json:"ip"`
	EnrollKey string `json:"enroll_key"`
}

type enrollResponse struct {
	DeviceID   string `json:"device_id"`
	AgentToken string `json:"agent_token"`
}

type heartbeatRequest struct {
	DeviceID   string `json:"device_id"`
	AgentToken string `json:"agent_token"`
}

// ---- Handlers ----

func (h *Handlers) Enroll(w http.ResponseWriter, r *http.Request) {
	var req enrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	dev := &Device{
		ID:         uuid.New().String(),
		Hostname:   req.Hostname,
		OS:         req.OS,
		IP:         req.IP,
		AgentToken: uuid.New().String(),
		EnrolledAt: time.Now().UTC(),
		LastSeen:   time.Now().UTC(),
	}
	if err := h.store.UpsertDevice(r.Context(), dev); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, enrollResponse{DeviceID: dev.ID, AgentToken: dev.AgentToken})
}

func (h *Handlers) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req heartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := h.store.UpdateLastSeen(r.Context(), req.DeviceID); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) GetPolicy(w http.ResponseWriter, r *http.Request) {
	policy := map[string]interface{}{
		"scan_interval_minutes": 60,
		"realtime_protection":   true,
		"log_level":             "info",
	}
	writeJSON(w, http.StatusOK, policy)
}

func (h *Handlers) IngestEvents(w http.ResponseWriter, r *http.Request) {
	var events []json.RawMessage
	_ = json.NewDecoder(r.Body).Decode(&events)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (h *Handlers) GetCommands(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []Command{})
}

func (h *Handlers) PostCommandResult(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "id")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) ListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.store.ListDevices(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, devices)
}

func (h *Handlers) GetDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	dev, err := h.store.GetDevice(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, dev)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
