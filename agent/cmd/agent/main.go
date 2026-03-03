package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	serverURL := getenv("SERVER_URL", "http://localhost:8080")
	enrollKey := os.Getenv("ENROLL_KEY")
	dataDir := getenv("DATA_DIR", "./data")

	logger := log.New(os.Stdout, "", log.LstdFlags)

	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		logger.Fatalf("create data dir: %v", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	stateFile := filepath.Join(dataDir, "state.json")

	state, err := loadState(stateFile)
	if err != nil {
		logger.Printf("no existing state, enrolling...")
		state, err = enroll(client, serverURL, enrollKey, logger)
		if err != nil {
			logger.Fatalf("enrollment failed: %v", err)
		}
		if err := saveState(stateFile, state); err != nil {
			logger.Fatalf("save state: %v", err)
		}
		logger.Printf("enrolled: device_id=%s", state.DeviceID)
	} else {
		logger.Printf("loaded existing state: device_id=%s", state.DeviceID)
	}

	// Heartbeat every 30s
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			if err := heartbeat(client, serverURL, state, logger); err != nil {
				logger.Printf("heartbeat error: %v", err)
			}
		}
	}()

	// Policy fetch every 5 min
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for range t.C {
			if err := fetchPolicy(client, serverURL, state, logger); err != nil {
				logger.Printf("policy fetch error: %v", err)
			}
		}
	}()

	// Command poll every 30s
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			if err := pollCommands(client, serverURL, state, logger); err != nil {
				logger.Printf("command poll error: %v", err)
			}
		}
	}()

	// Block forever
	select {}
}

// ---- State ----

type agentState struct {
	DeviceID   string `json:"device_id"`
	AgentToken string `json:"agent_token"`
}

func loadState(path string) (*agentState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s agentState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.DeviceID == "" || s.AgentToken == "" {
		return nil, fmt.Errorf("invalid state")
	}
	return &s, nil
}

func saveState(path string, s *agentState) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// ---- Enroll ----

func enroll(client *http.Client, serverURL, enrollKey string, _ *log.Logger) (*agentState, error) {
	hostname, _ := os.Hostname()
	ip := localIP()

	body := map[string]string{
		"hostname":   hostname,
		"os":         "windows",
		"ip":         ip,
		"enroll_key": enrollKey,
	}
	resp, err := postJSON(client, serverURL+"/api/agent/enroll", "", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("enroll HTTP %d: %s", resp.StatusCode, string(b))
	}

	var result struct {
		DeviceID   string `json:"device_id"`
		AgentToken string `json:"agent_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode enroll response: %w", err)
	}
	return &agentState{DeviceID: result.DeviceID, AgentToken: result.AgentToken}, nil
}

// ---- Heartbeat ----

func heartbeat(client *http.Client, serverURL string, state *agentState, logger *log.Logger) error {
	body := map[string]string{
		"device_id":   state.DeviceID,
		"agent_token": state.AgentToken,
	}
	resp, err := postJSON(client, serverURL+"/api/agent/heartbeat", state.AgentToken, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat HTTP %d: %s", resp.StatusCode, string(b))
	}
	logger.Printf("heartbeat ok")
	return nil
}

// ---- Policy ----

func fetchPolicy(client *http.Client, serverURL string, state *agentState, logger *log.Logger) error {
	req, err := http.NewRequest(http.MethodGet, serverURL+"/api/agent/policy", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+state.AgentToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var policy map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return err
	}
	logger.Printf("policy: %v", policy)
	return nil
}

// ---- Commands ----

type command struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func pollCommands(client *http.Client, serverURL string, state *agentState, logger *log.Logger) error {
	req, err := http.NewRequest(http.MethodGet, serverURL+"/api/agent/commands", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+state.AgentToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var commands []command
	if err := json.NewDecoder(resp.Body).Decode(&commands); err != nil {
		return err
	}
	for _, cmd := range commands {
		logger.Printf("received command id=%s type=%s payload=%s", cmd.ID, cmd.Type, string(cmd.Payload))
	}
	return nil
}

// ---- Helpers ----

func postJSON(client *http.Client, url, token string, body interface{}) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return client.Do(req)
}

func localIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "unknown"
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
