package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/teecert/openclaw-configurator/internal/config"
	"github.com/teecert/openclaw-configurator/internal/connection"
	"github.com/teecert/openclaw-configurator/internal/detector"
)

type AppState struct {
	FS          connection.FileSystem
	ConfigPath  string
	Config      *config.OpenClawConfig
	TargetOS    string
	Mode        string
	RecentPaths []string
}

var state = &AppState{}

var providerNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Mode     string                   `json:"mode"`
		OS       string                   `json:"os"`
		SSH      *connection.SSHConfig    `json:"ssh,omitempty"`
		Docker   *connection.DockerConfig `json:"docker,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.OS == "" {
		req.OS = "linux"
	}
	if req.Mode != "local" && req.Mode != "remote" && req.Mode != "docker" {
		jsonError(w, "mode must be 'local', 'remote', or 'docker'", http.StatusBadRequest)
		return
	}

	if state.FS != nil {
		state.FS.Close()
		state.FS = nil
	}
	state.Config = nil
	state.ConfigPath = ""

	state.TargetOS = req.OS
	state.Mode = req.Mode

	if req.Mode == "local" {
		state.FS = connection.NewLocalFS()
		jsonResponse(w, map[string]interface{}{"ok": true, "mode": "local"})
		return
	}

	if req.Mode == "docker" {
		if req.Docker == nil {
			jsonError(w, "docker config required for docker mode", http.StatusBadRequest)
			return
		}
		dockerFS, err := connection.NewDockerFS(*req.Docker)
		if err != nil {
			jsonError(w, fmt.Sprintf("Docker connection failed: %s", err.Error()), http.StatusBadRequest)
			return
		}
		state.FS = dockerFS
		jsonResponse(w, map[string]interface{}{"ok": true, "mode": "docker"})
		return
	}

	if req.SSH == nil {
		jsonError(w, "ssh config required for remote mode", http.StatusBadRequest)
		return
	}
	sshFS, err := connection.NewSSHFS(*req.SSH)
	if err != nil {
		jsonError(w, fmt.Sprintf("SSH connection failed: %s", err.Error()), http.StatusBadRequest)
		return
	}
	state.FS = sshFS
	jsonResponse(w, map[string]interface{}{"ok": true, "mode": "remote"})
}

func handleDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state.FS != nil {
		state.FS.Close()
		state.FS = nil
	}
	state.Config = nil
	state.ConfigPath = ""
	jsonResponse(w, map[string]interface{}{"ok": true})
}

func handleDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state.FS == nil {
		jsonError(w, "not connected", http.StatusBadRequest)
		return
	}

	detectedPath, found, err := detector.DetectOpenClawPath(state.FS, state.TargetOS)
	if err != nil {
		jsonError(w, "detection failed", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"path":        detectedPath,
		"found":       found,
		"recentPaths": state.RecentPaths,
	})
}

func handleSetPath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state.FS == nil {
		jsonError(w, "not connected", http.StatusBadRequest)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		jsonError(w, "path is required", http.StatusBadRequest)
		return
	}
	if !strings.HasSuffix(strings.ToLower(req.Path), ".json") {
		jsonError(w, "path must point to a JSON file", http.StatusBadRequest)
		return
	}

	info, err := state.FS.Stat(req.Path)
	if err != nil {
		jsonError(w, "path does not exist", http.StatusBadRequest)
		return
	}
	if info.IsDir {
		jsonError(w, "path is a directory, expected a file", http.StatusBadRequest)
		return
	}

	state.ConfigPath = req.Path
	cfg, err := config.ReadConfig(state.FS, req.Path)
	if err != nil {
		jsonError(w, fmt.Sprintf("failed to read config: %s", err.Error()), http.StatusBadRequest)
		return
	}
	state.Config = cfg

	alreadyTracked := false
	for _, p := range state.RecentPaths {
		if p == req.Path {
			alreadyTracked = true
			break
		}
	}
	if !alreadyTracked {
		state.RecentPaths = append(state.RecentPaths, req.Path)
		if len(state.RecentPaths) > 10 {
			state.RecentPaths = state.RecentPaths[len(state.RecentPaths)-10:]
		}
	}

	jsonResponse(w, map[string]interface{}{"ok": true, "path": req.Path})
}

func handleGetModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state.Config == nil {
		jsonError(w, "config not loaded", http.StatusBadRequest)
		return
	}

	masked := make(map[string]*providerView)
	for name, prov := range state.Config.Models.Providers {
		masked[name] = &providerView{
			BaseURL:   prov.BaseURL,
			APIKey:    config.MaskAPIKey(prov.APIKey),
			API:       prov.API,
			Models:    prov.Models,
			HasAPIKey: prov.APIKey != "",
		}
	}

	jsonResponse(w, map[string]interface{}{
		"mode":      state.Config.Models.Mode,
		"providers": masked,
		"primary":   state.Config.Agents.Defaults.Model.Primary,
	})
}

type providerView struct {
	BaseURL   string         `json:"baseUrl"`
	APIKey    string         `json:"apiKey"`
	API       string         `json:"api"`
	Models    []config.Model `json:"models"`
	HasAPIKey bool           `json:"hasApiKey"`
}

func handleGetAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state.Config == nil {
		jsonError(w, "config not loaded", http.StatusBadRequest)
		return
	}

	statuses, err := config.GetAgentStatuses(state.FS, state.ConfigPath, state.Config)
	if err != nil {
		jsonError(w, "failed to get agent statuses", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]interface{}{"agents": statuses})
}

func handleProviders(w http.ResponseWriter, r *http.Request) {
	if state.Config == nil {
		jsonError(w, "config not loaded", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		handleAddProvider(w, r)
	default:
		path := strings.TrimPrefix(r.URL.Path, "/api/providers/")
		if path == "" || path == r.URL.Path {
			jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		parts := strings.SplitN(path, "/", 3)
		provName := parts[0]

		if len(parts) == 1 {
			switch r.Method {
			case http.MethodPut:
				handleUpdateProvider(w, r, provName)
			case http.MethodDelete:
				handleDeleteProvider(w, r, provName)
			default:
				jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		if len(parts) >= 2 && parts[1] == "models" {
			if len(parts) == 2 {
				if r.Method == http.MethodPost {
					handleAddModel(w, r, provName)
				} else {
					jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}
			modelID := parts[2]
			switch r.Method {
			case http.MethodPut:
				handleUpdateModel(w, r, provName, modelID)
			case http.MethodDelete:
				handleDeleteModel(w, r, provName, modelID)
			default:
				jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		jsonError(w, "not found", http.StatusNotFound)
	}
}

func handleAddProvider(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		BaseURL      string `json:"baseUrl"`
		APIKey       string `json:"apiKey"`
		API          string `json:"api"`
		DefaultModel string `json:"defaultModel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	name := req.Name
	if !strings.HasPrefix(name, "custom-") {
		name = "custom-" + name
	}
	if !providerNameRe.MatchString(name) {
		jsonError(w, "provider name must match [a-z0-9-]", http.StatusBadRequest)
		return
	}
	if !isValidURL(req.BaseURL) {
		jsonError(w, "baseUrl must start with http:// or https://", http.StatusBadRequest)
		return
	}
	if !isValidAPI(req.API) {
		jsonError(w, "invalid api type", http.StatusBadRequest)
		return
	}
	if req.DefaultModel == "" {
		jsonError(w, "a default model ID is required", http.StatusBadRequest)
		return
	}

	models := []config.Model{{
		ID:            req.DefaultModel,
		Name:          req.DefaultModel + " (Custom Provider)",
		API:           req.API,
		Reasoning:     false,
		Input:         []string{"text"},
		Cost:          config.Cost{},
		ContextWindow: 128000,
		MaxTokens:     8192,
	}}

	prov := &config.Provider{
		BaseURL: req.BaseURL,
		APIKey:  req.APIKey,
		API:     req.API,
		Models:  models,
	}
	if err := config.AddProvider(state.Config, name, prov); err != nil {
		jsonError(w, err.Error(), http.StatusConflict)
		return
	}
	jsonResponse(w, map[string]interface{}{"ok": true, "name": name})
}

func handleUpdateProvider(w http.ResponseWriter, r *http.Request, name string) {
	var req struct {
		BaseURL string `json:"baseUrl"`
		APIKey  string `json:"apiKey"`
		API     string `json:"api"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	existing, ok := state.Config.Models.Providers[name]
	if !ok {
		jsonError(w, "provider not found", http.StatusNotFound)
		return
	}

	if req.BaseURL != "" {
		if !isValidURL(req.BaseURL) {
			jsonError(w, "baseUrl must start with http:// or https://", http.StatusBadRequest)
			return
		}
		existing.BaseURL = req.BaseURL
	}
	if req.APIKey != "" && req.APIKey != config.MaskAPIKey(existing.APIKey) {
		existing.APIKey = req.APIKey
	}
	if req.API != "" {
		if !isValidAPI(req.API) {
			jsonError(w, "invalid api type", http.StatusBadRequest)
			return
		}
		existing.API = req.API
	}

	config.SyncDefaultModels(state.Config)
	jsonResponse(w, map[string]interface{}{"ok": true})
}

func handleDeleteProvider(w http.ResponseWriter, r *http.Request, name string) {
	if err := config.DeleteProvider(state.Config, name); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonResponse(w, map[string]interface{}{"ok": true})
}

func handleAddModel(w http.ResponseWriter, r *http.Request, provName string) {
	var model config.Model
	if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if model.ID == "" {
		jsonError(w, "model id is required", http.StatusBadRequest)
		return
	}
	if model.Name == "" {
		model.Name = model.ID + " (Custom Provider)"
	}
	if model.Input == nil {
		model.Input = []string{"text"}
	}
	if model.ContextWindow <= 0 {
		model.ContextWindow = 200000
	}
	if model.MaxTokens <= 0 {
		model.MaxTokens = 16384
	}

	prov, ok := state.Config.Models.Providers[provName]
	if ok && model.API == "" {
		model.API = prov.API
	}

	if err := config.AddModel(state.Config, provName, model); err != nil {
		jsonError(w, err.Error(), http.StatusConflict)
		return
	}
	jsonResponse(w, map[string]interface{}{"ok": true})
}

func handleUpdateModel(w http.ResponseWriter, r *http.Request, provName, modelID string) {
	var model config.Model
	if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if model.ID == "" {
		model.ID = modelID
	}
	if err := config.UpdateModel(state.Config, provName, modelID, model); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonResponse(w, map[string]interface{}{"ok": true})
}

func handleDeleteModel(w http.ResponseWriter, r *http.Request, provName, modelID string) {
	prov, ok := state.Config.Models.Providers[provName]
	if !ok {
		jsonError(w, fmt.Sprintf("provider %q not found", provName), http.StatusNotFound)
		return
	}
	if len(prov.Models) <= 1 {
		jsonError(w, "cannot delete the last model — each provider must have at least one model", http.StatusBadRequest)
		return
	}
	if err := config.DeleteModel(state.Config, provName, modelID); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonResponse(w, map[string]interface{}{"ok": true})
}

func handleRawConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetRawConfig(w, r)
	case http.MethodPut:
		handlePutRawConfig(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetRawConfig(w http.ResponseWriter, r *http.Request) {
	if state.FS == nil || state.ConfigPath == "" {
		jsonError(w, "config not loaded", http.StatusBadRequest)
		return
	}
	data, err := state.FS.ReadFile(state.ConfigPath)
	if err != nil {
		jsonError(w, "failed to read config file", http.StatusInternalServerError)
		return
	}

	maskedData, err := config.MaskRawJSON(data)
	if err != nil {
		jsonError(w, "failed to process config file", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]interface{}{"content": string(maskedData)})
}

func handlePutRawConfig(w http.ResponseWriter, r *http.Request) {
	if state.FS == nil || state.ConfigPath == "" {
		jsonError(w, "config not loaded", http.StatusBadRequest)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	newBytes := []byte(req.Content)

	origData, readErr := state.FS.ReadFile(state.ConfigPath)
	if readErr == nil {
		restored, err := config.RestoreRawJSON(newBytes, origData)
		if err != nil {
			jsonError(w, fmt.Sprintf("JSON parse error: %s", err.Error()), http.StatusBadRequest)
			return
		}
		newBytes = restored
	}

	var newCfg config.OpenClawConfig
	if err := json.Unmarshal(newBytes, &newCfg); err != nil {
		jsonError(w, fmt.Sprintf("JSON parse error: %s", err.Error()), http.StatusBadRequest)
		return
	}

	if err := config.WriteRawConfig(state.FS, state.ConfigPath, newBytes); err != nil {
		jsonError(w, fmt.Sprintf("failed to save: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	state.Config = &newCfg
	config.SyncDefaultModels(state.Config)
	jsonResponse(w, map[string]interface{}{"ok": true})
}

func handleSetPrimary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state.Config == nil {
		jsonError(w, "config not loaded", http.StatusBadRequest)
		return
	}

	var req struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Model == "" {
		jsonError(w, "model reference is required (format: provider/model-id)", http.StatusBadRequest)
		return
	}

	if _, ok := state.Config.Agents.Defaults.Models[req.Model]; !ok {
		jsonError(w, "model not found in available models", http.StatusBadRequest)
		return
	}

	state.Config.Agents.Defaults.Model.Primary = req.Model
	jsonResponse(w, map[string]interface{}{"ok": true, "primary": req.Model})
}

func handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state.Config == nil || state.ConfigPath == "" {
		jsonError(w, "config not loaded", http.StatusBadRequest)
		return
	}

	if err := config.WriteConfig(state.FS, state.ConfigPath, state.Config); err != nil {
		jsonError(w, fmt.Sprintf("failed to save: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	synced, err := config.SyncAgentModels(state.FS, state.ConfigPath, state.Config)
	if err != nil {
		jsonResponse(w, map[string]interface{}{
			"ok":     true,
			"saved":  true,
			"synced": []string{},
			"warn":   "config saved but agent sync had issues",
		})
		return
	}

	jsonResponse(w, map[string]interface{}{
		"ok":     true,
		"saved":  true,
		"synced": synced,
	})
}

func isValidURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

func isValidAPI(api string) bool {
	valid := map[string]bool{
		"openai-completions":     true,
		"anthropic-messages":     true,
		"openai-codex-responses": true,
	}
	return valid[api]
}

func handleTestProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state.Config == nil {
		jsonError(w, "config not loaded", http.StatusBadRequest)
		return
	}
	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	prov, ok := state.Config.Models.Providers[req.Provider]
	if !ok {
		jsonError(w, "provider not found", http.StatusNotFound)
		return
	}

	start := time.Now()
	result := testAPIConnection(prov.BaseURL, prov.APIKey, prov.API)
	latency := time.Since(start).Milliseconds()

	jsonResponse(w, map[string]interface{}{
		"ok":        result.ok,
		"status":    result.status,
		"message":   result.message,
		"latencyMs": latency,
	})
}

type apiTestResult struct {
	ok      bool
	status  int
	message string
}

func testAPIConnection(baseURL, apiKey, apiType string) apiTestResult {
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Strip auth headers on redirect to prevent key leakage
			req.Header.Del("Authorization")
			req.Header.Del("x-api-key")
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	testURL := strings.TrimRight(baseURL, "/") + "/models"

	testReq, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return apiTestResult{ok: false, message: "Invalid URL"}
	}
	switch apiType {
	case "anthropic-messages":
		testReq.Header.Set("x-api-key", apiKey)
		testReq.Header.Set("anthropic-version", "2023-06-01")
	default:
		testReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(testReq)
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "no such host"):
			return apiTestResult{ok: false, message: "Host not found"}
		case strings.Contains(msg, "connection refused"):
			return apiTestResult{ok: false, message: "Connection refused"}
		case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded"):
			return apiTestResult{ok: false, message: "Connection timeout"}
		default:
			return apiTestResult{ok: false, message: "Connection failed"}
		}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return apiTestResult{ok: true, status: resp.StatusCode, message: "OK"}
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return apiTestResult{ok: false, status: resp.StatusCode, message: "Invalid API key"}
	}
	if resp.StatusCode == 404 {
		return apiTestResult{ok: false, status: 404, message: "Endpoint not found"}
	}
	return apiTestResult{ok: false, status: resp.StatusCode, message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state.FS == nil || state.ConfigPath == "" {
		jsonError(w, "config not loaded", http.StatusBadRequest)
		return
	}
	data, err := state.FS.ReadFile(state.ConfigPath)
	if err != nil {
		jsonError(w, "failed to read config", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="openclaw.json"`)
	w.Write(data)
}

func handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state.FS == nil || state.ConfigPath == "" {
		jsonError(w, "config not loaded", http.StatusBadRequest)
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	newBytes := []byte(req.Content)
	var newCfg config.OpenClawConfig
	if err := json.Unmarshal(newBytes, &newCfg); err != nil {
		jsonError(w, fmt.Sprintf("Invalid JSON: %s", err.Error()), http.StatusBadRequest)
		return
	}

	var generic interface{}
	if json.Unmarshal(newBytes, &generic) == nil {
		if pretty, err := json.MarshalIndent(generic, "", "  "); err == nil {
			newBytes = append(pretty, '\n')
		}
	}

	if err := config.WriteRawConfig(state.FS, state.ConfigPath, newBytes); err != nil {
		jsonError(w, fmt.Sprintf("Import failed: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	state.Config = &newCfg
	config.SyncDefaultModels(state.Config)
	jsonResponse(w, map[string]interface{}{"ok": true})
}

func handleDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if state.FS == nil || state.ConfigPath == "" || state.Config == nil {
		jsonError(w, "config not loaded", http.StatusBadRequest)
		return
	}
	diskData, err := state.FS.ReadFile(state.ConfigPath)
	if err != nil {
		jsonError(w, "failed to read config", http.StatusInternalServerError)
		return
	}
	maskedDisk, _ := config.MaskRawJSON(diskData)
	memData, _ := json.MarshalIndent(state.Config, "", "  ")
	maskedMem, _ := config.MaskRawJSON(memData)

	jsonResponse(w, map[string]interface{}{
		"disk":   string(maskedDisk),
		"memory": string(maskedMem),
	})
}
