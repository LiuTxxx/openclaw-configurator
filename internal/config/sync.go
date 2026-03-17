package config

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/teecert/openclaw-configurator/internal/connection"
)

func SyncAgentModels(cfs connection.FileSystem, configPath string, cfg *OpenClawConfig) ([]string, error) {
	SyncDefaultModels(cfg)

	normConfig := strings.ReplaceAll(configPath, "\\", "/")
	home, _ := cfs.HomeDir()
	normHome := strings.ReplaceAll(home, "\\", "/")
	openclawDir := path.Dir(normConfig)
	agentsDir := path.Join(openclawDir, "agents")

	var synced []string
	for _, agent := range cfg.Agents.List {
		agentDir := strings.ReplaceAll(agent.AgentDir, "\\", "/")
		if agentDir == "" {
			agentDir = path.Join(agentsDir, agent.ID, "agent")
		}
		if normHome != "" && strings.HasPrefix(agentDir, "~") {
			agentDir = normHome + agentDir[1:]
		}
		agentDir = path.Clean(agentDir)

		modelsPath := path.Join(agentDir, "models.json")
		if _, err := cfs.Stat(modelsPath); err == nil {
			if err := cfs.Remove(modelsPath); err != nil {
				continue
			}
			synced = append(synced, agent.ID)
		}
	}
	return synced, nil
}

func SyncDefaultModels(cfg *OpenClawConfig) {
	newModels := make(map[string]json.RawMessage)
	for provName, prov := range cfg.Models.Providers {
		for _, model := range prov.Models {
			ref := provName + "/" + model.ID
			newModels[ref] = json.RawMessage("{}")
		}
	}
	cfg.Agents.Defaults.Models = newModels

	if cfg.Agents.Defaults.Model.Primary != "" {
		if _, ok := newModels[cfg.Agents.Defaults.Model.Primary]; !ok {
			for ref := range newModels {
				cfg.Agents.Defaults.Model.Primary = ref
				break
			}
			if len(newModels) == 0 {
				cfg.Agents.Defaults.Model.Primary = ""
			}
		}
	}
}

type AgentStatus struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	AgentDir       string `json:"agentDir"`
	HasModelsJSON  bool   `json:"hasModelsJson"`
	ModelsJSONPath string `json:"modelsJsonPath"`
}

func GetAgentStatuses(cfs connection.FileSystem, configPath string, cfg *OpenClawConfig) ([]AgentStatus, error) {
	normConfig := strings.ReplaceAll(configPath, "\\", "/")
	home, _ := cfs.HomeDir()
	normHome := strings.ReplaceAll(home, "\\", "/")
	openclawDir := path.Dir(normConfig)
	agentsDir := path.Join(openclawDir, "agents")

	var statuses []AgentStatus
	for _, agent := range cfg.Agents.List {
		agentDir := strings.ReplaceAll(agent.AgentDir, "\\", "/")
		if agentDir == "" {
			agentDir = path.Join(agentsDir, agent.ID, "agent")
		}
		if normHome != "" && strings.HasPrefix(agentDir, "~") {
			agentDir = normHome + agentDir[1:]
		}
		agentDir = path.Clean(agentDir)

		modelsPath := path.Join(agentDir, "models.json")
		_, err := cfs.Stat(modelsPath)
		name := agent.Name
		if name == "" {
			name = agent.ID
		}

		statuses = append(statuses, AgentStatus{
			ID:             agent.ID,
			Name:           name,
			AgentDir:       agentDir,
			HasModelsJSON:  err == nil,
			ModelsJSONPath: modelsPath,
		})
	}
	return statuses, nil
}

func AddProvider(cfg *OpenClawConfig, name string, prov *Provider) error {
	if cfg.Models.Providers == nil {
		cfg.Models.Providers = make(map[string]*Provider)
	}
	if _, exists := cfg.Models.Providers[name]; exists {
		return fmt.Errorf("provider %q already exists", name)
	}
	cfg.Models.Providers[name] = prov
	SyncDefaultModels(cfg)
	return nil
}

func UpdateProvider(cfg *OpenClawConfig, name string, prov *Provider) error {
	if _, exists := cfg.Models.Providers[name]; !exists {
		return fmt.Errorf("provider %q not found", name)
	}
	cfg.Models.Providers[name] = prov
	SyncDefaultModels(cfg)
	return nil
}

func DeleteProvider(cfg *OpenClawConfig, name string) error {
	if _, exists := cfg.Models.Providers[name]; !exists {
		return fmt.Errorf("provider %q not found", name)
	}
	delete(cfg.Models.Providers, name)
	SyncDefaultModels(cfg)
	return nil
}

func AddModel(cfg *OpenClawConfig, provName string, model Model) error {
	prov, ok := cfg.Models.Providers[provName]
	if !ok {
		return fmt.Errorf("provider %q not found", provName)
	}
	for _, m := range prov.Models {
		if m.ID == model.ID {
			return fmt.Errorf("model %q already exists in provider %q", model.ID, provName)
		}
	}
	prov.Models = append(prov.Models, model)
	SyncDefaultModels(cfg)
	return nil
}

func UpdateModel(cfg *OpenClawConfig, provName, modelID string, model Model) error {
	prov, ok := cfg.Models.Providers[provName]
	if !ok {
		return fmt.Errorf("provider %q not found", provName)
	}
	for i, m := range prov.Models {
		if m.ID == modelID {
			prov.Models[i] = model
			SyncDefaultModels(cfg)
			return nil
		}
	}
	return fmt.Errorf("model %q not found in provider %q", modelID, provName)
}

func DeleteModel(cfg *OpenClawConfig, provName, modelID string) error {
	prov, ok := cfg.Models.Providers[provName]
	if !ok {
		return fmt.Errorf("provider %q not found", provName)
	}
	for i, m := range prov.Models {
		if m.ID == modelID {
			prov.Models = append(prov.Models[:i], prov.Models[i+1:]...)
			SyncDefaultModels(cfg)
			return nil
		}
	}
	return fmt.Errorf("model %q not found in provider %q", modelID, provName)
}
