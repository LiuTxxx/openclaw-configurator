package config

import "encoding/json"

type OpenClawConfig struct {
	Meta     json.RawMessage `json:"meta,omitempty"`
	Wizard   json.RawMessage `json:"wizard,omitempty"`
	Browser  json.RawMessage `json:"browser,omitempty"`
	Auth     json.RawMessage `json:"auth,omitempty"`
	Models   ModelsConfig    `json:"models"`
	Agents   AgentsConfig    `json:"agents"`
	Bindings json.RawMessage `json:"bindings,omitempty"`
	Messages json.RawMessage `json:"messages,omitempty"`
	Commands json.RawMessage `json:"commands,omitempty"`
	Session  json.RawMessage `json:"session,omitempty"`
	Hooks    json.RawMessage `json:"hooks,omitempty"`
	Channels json.RawMessage `json:"channels,omitempty"`
	Gateway  json.RawMessage `json:"gateway,omitempty"`
	Plugins  json.RawMessage `json:"plugins,omitempty"`

	extra map[string]json.RawMessage
}

func (c *OpenClawConfig) UnmarshalJSON(data []byte) error {
	type Alias OpenClawConfig
	aux := &struct {
		*Alias
	}{Alias: (*Alias)(c)}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	known := map[string]bool{
		"meta": true, "wizard": true, "browser": true, "auth": true,
		"models": true, "agents": true, "bindings": true, "messages": true,
		"commands": true, "session": true, "hooks": true, "channels": true,
		"gateway": true, "plugins": true,
	}
	c.extra = make(map[string]json.RawMessage)
	for k, v := range raw {
		if !known[k] {
			c.extra[k] = v
		}
	}
	return nil
}

func (c OpenClawConfig) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})

	setRaw := func(key string, v json.RawMessage) {
		if len(v) > 0 {
			m[key] = v
		}
	}
	setRaw("meta", c.Meta)
	setRaw("wizard", c.Wizard)
	setRaw("browser", c.Browser)
	setRaw("auth", c.Auth)
	m["models"] = c.Models
	m["agents"] = c.Agents
	setRaw("bindings", c.Bindings)
	setRaw("messages", c.Messages)
	setRaw("commands", c.Commands)
	setRaw("session", c.Session)
	setRaw("hooks", c.Hooks)
	setRaw("channels", c.Channels)
	setRaw("gateway", c.Gateway)
	setRaw("plugins", c.Plugins)

	for k, v := range c.extra {
		m[k] = v
	}
	return json.Marshal(m)
}

type ModelsConfig struct {
	Mode      string               `json:"mode,omitempty"`
	Providers map[string]*Provider `json:"providers"`
}

type Provider struct {
	BaseURL string  `json:"baseUrl"`
	APIKey  string  `json:"apiKey,omitempty"`
	API     string  `json:"api"`
	Models  []Model `json:"models"`
}

type Model struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	API           string   `json:"api,omitempty"`
	Reasoning     bool     `json:"reasoning"`
	Input         []string `json:"input"`
	Cost          Cost     `json:"cost"`
	ContextWindow int      `json:"contextWindow"`
	MaxTokens     int      `json:"maxTokens"`
}

type Cost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
	List     []AgentEntry  `json:"list"`
}

type AgentDefaults struct {
	Model    PrimaryModel                   `json:"model"`
	Models   map[string]json.RawMessage     `json:"models"`
	Extra    map[string]json.RawMessage     `json:"-"`
}

func (d *AgentDefaults) UnmarshalJSON(data []byte) error {
	type Alias AgentDefaults
	aux := &struct {
		*Alias
	}{Alias: (*Alias)(d)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	known := map[string]bool{"model": true, "models": true}
	d.Extra = make(map[string]json.RawMessage)
	for k, v := range raw {
		if !known[k] {
			d.Extra[k] = v
		}
	}
	return nil
}

func (d AgentDefaults) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["model"] = d.Model
	m["models"] = d.Models
	for k, v := range d.Extra {
		m[k] = v
	}
	return json.Marshal(m)
}

type PrimaryModel struct {
	Primary string `json:"primary"`
}

type AgentEntry struct {
	ID       string          `json:"id"`
	Name     string          `json:"name,omitempty"`
	AgentDir string          `json:"agentDir,omitempty"`
	Extra    json.RawMessage `json:"-"`
}

func (a *AgentEntry) UnmarshalJSON(data []byte) error {
	type Alias AgentEntry
	aux := &struct {
		*Alias
	}{Alias: (*Alias)(a)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	known := map[string]bool{"id": true, "name": true, "agentDir": true}
	extra := make(map[string]json.RawMessage)
	for k, v := range raw {
		if !known[k] {
			extra[k] = v
		}
	}
	if len(extra) > 0 {
		a.Extra, _ = json.Marshal(extra)
	}
	return nil
}

func (a AgentEntry) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["id"] = a.ID
	if a.Name != "" {
		m["name"] = a.Name
	}
	if a.AgentDir != "" {
		m["agentDir"] = a.AgentDir
	}
	if len(a.Extra) > 0 {
		var extra map[string]json.RawMessage
		if err := json.Unmarshal(a.Extra, &extra); err == nil {
			for k, v := range extra {
				m[k] = v
			}
		}
	}
	return json.Marshal(m)
}

type ModelsResponse struct {
	Mode      string                `json:"mode"`
	Providers map[string]*Provider  `json:"providers"`
	Primary   string                `json:"primary"`
	ModelRefs map[string]struct{}   `json:"modelRefs"`
}

func MaskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-3:]
}
