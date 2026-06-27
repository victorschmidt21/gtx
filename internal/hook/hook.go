package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
)

const gtxHookCommand = "gtx rewrite"
const backupSuffix = ".gtx-backup"

// HookInstaller gerencia a instalação do hook GTX no Claude Code.
type HookInstaller interface {
	SettingsPath() (string, error)
	Install() error
	Uninstall() error
	Verify() (bool, error)
}

// New cria o installer adequado para o OS atual.
func New() HookInstaller {
	if runtime.GOOS == "windows" {
		return &windowsHook{}
	}
	return &unixHook{}
}

// settings é a estrutura mínima do settings.json do Claude Code.
type settings struct {
	Hooks map[string][]hookEntry `json:"hooks,omitempty"`
	Other map[string]interface{} `json:"-"`
}

type hookEntry struct {
	Matcher string      `json:"matcher,omitempty"`
	Hooks   []hookDef   `json:"hooks,omitempty"`
}

type hookDef struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

func loadSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]interface{}{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("settings.json inválido: %w", err)
	}
	return m, nil
}

func saveSettings(path string, m map[string]interface{}) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func backupSettings(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.WriteFile(path+backupSuffix, data, 0644)
}

// isGtxHookPresent verifica se o hook GTX já está no array de hooks.
func isGtxHookPresent(m map[string]interface{}) bool {
	hooks, ok := m["hooks"].(map[string]interface{})
	if !ok {
		return false
	}
	preToolUse, ok := hooks["PreToolUse"]
	if !ok {
		return false
	}
	entries, ok := preToolUse.([]interface{})
	if !ok {
		return false
	}
	for _, entry := range entries {
		e, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		matcher, _ := e["matcher"].(string)
		if matcher != "Bash" && matcher != "PowerShell" {
			continue
		}
		subHooks, ok := e["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, sh := range subHooks {
			shm, ok := sh.(map[string]interface{})
			if !ok {
				continue
			}
			if cmd, _ := shm["command"].(string); cmd == gtxHookCommand {
				return true
			}
		}
	}
	return false
}

func installHook(path string) error {
	if err := backupSettings(path); err != nil {
		return fmt.Errorf("backup falhou: %w", err)
	}

	m, err := loadSettings(path)
	if err != nil {
		return err
	}

	if isGtxHookPresent(m) {
		fmt.Println("hook GTX já está instalado")
		return nil
	}

	// garante que hooks e PreToolUse existem
	hooksMap, ok := m["hooks"].(map[string]interface{})
	if !ok {
		hooksMap = map[string]interface{}{}
	}

	newEntry := map[string]interface{}{
		"matcher": "Bash",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": gtxHookCommand,
			},
		},
	}

	preToolUse, ok := hooksMap["PreToolUse"].([]interface{})
	if !ok {
		preToolUse = []interface{}{}
	}
	preToolUse = append(preToolUse, newEntry)
	hooksMap["PreToolUse"] = preToolUse
	m["hooks"] = hooksMap

	if err := os.MkdirAll(pathDir(path), 0755); err != nil {
		return err
	}
	return saveSettings(path, m)
}

func uninstallHook(path string) error {
	m, err := loadSettings(path)
	if err != nil {
		return err
	}

	if !isGtxHookPresent(m) {
		fmt.Println("hook GTX não está instalado")
		return nil
	}

	hooksMap, ok := m["hooks"].(map[string]interface{})
	if !ok {
		return nil
	}
	preToolUse, ok := hooksMap["PreToolUse"].([]interface{})
	if !ok {
		return nil
	}

	var filtered []interface{}
	for _, entry := range preToolUse {
		e, ok := entry.(map[string]interface{})
		if !ok {
			filtered = append(filtered, entry)
			continue
		}
		subHooks, ok := e["hooks"].([]interface{})
		if !ok {
			filtered = append(filtered, entry)
			continue
		}
		var newSubs []interface{}
		for _, sh := range subHooks {
			shm, ok := sh.(map[string]interface{})
			if !ok || shm["command"] != gtxHookCommand {
				newSubs = append(newSubs, sh)
			}
		}
		if len(newSubs) > 0 {
			e["hooks"] = newSubs
			filtered = append(filtered, e)
		}
	}

	hooksMap["PreToolUse"] = filtered
	m["hooks"] = hooksMap
	return saveSettings(path, m)
}

func verifyHook(path string) (bool, error) {
	m, err := loadSettings(path)
	if err != nil {
		return false, err
	}
	return isGtxHookPresent(m), nil
}

func pathDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
