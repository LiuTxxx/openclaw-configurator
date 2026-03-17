package detector

import (
	"path"
	"strings"

	"github.com/teecert/openclaw-configurator/internal/connection"
)

func DetectOpenClawPath(fs connection.FileSystem, targetOS string) (string, bool, error) {
	if envPath := fs.GetEnv("OPENCLAW_CONFIG_PATH"); envPath != "" {
		envPath = strings.TrimSpace(envPath)
		info, err := fs.Stat(envPath)
		if err == nil && !info.IsDir {
			return envPath, true, nil
		}
	}

	home, err := fs.HomeDir()
	if err != nil {
		return "", false, err
	}

	if envHome := fs.GetEnv("OPENCLAW_HOME"); envHome != "" {
		envHome = strings.TrimSpace(envHome)
		var p string
		if targetOS == "windows" {
			p = envHome + "\\openclaw.json"
		} else {
			p = path.Join(envHome, "openclaw.json")
		}
		info, err := fs.Stat(p)
		if err == nil && !info.IsDir {
			return p, true, nil
		}
	}

	if envState := fs.GetEnv("OPENCLAW_STATE_DIR"); envState != "" {
		envState = strings.TrimSpace(envState)
		var p string
		if targetOS == "windows" {
			p = envState + "\\openclaw.json"
		} else {
			p = path.Join(envState, "openclaw.json")
		}
		info, err := fs.Stat(p)
		if err == nil && !info.IsDir {
			return p, true, nil
		}
	}

	var candidates []string
	switch targetOS {
	case "windows":
		candidates = []string{
			home + "\\.openclaw\\openclaw.json",
			home + "\\AppData\\Local\\openclaw\\openclaw.json",
		}
	default:
		candidates = []string{
			path.Join(home, ".openclaw", "openclaw.json"),
		}
	}

	for _, p := range candidates {
		info, err := fs.Stat(p)
		if err == nil && !info.IsDir {
			return p, true, nil
		}
	}

	defaultPath := candidates[0]
	return defaultPath, false, nil
}
