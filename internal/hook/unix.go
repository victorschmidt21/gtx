package hook

import (
	"fmt"
	"os"
	"path/filepath"
)

type unixHook struct{}

func (h *unixHook) SettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível determinar home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func (h *unixHook) Install() error {
	path, err := h.SettingsPath()
	if err != nil {
		return err
	}
	if err := installHook(path); err != nil {
		return err
	}
	fmt.Printf("hook GTX instalado em %s\n", path)
	return nil
}

func (h *unixHook) Uninstall() error {
	path, err := h.SettingsPath()
	if err != nil {
		return err
	}
	if err := uninstallHook(path); err != nil {
		return err
	}
	fmt.Printf("hook GTX removido de %s\n", path)
	return nil
}

func (h *unixHook) Verify() (bool, error) {
	path, err := h.SettingsPath()
	if err != nil {
		return false, err
	}
	return verifyHook(path)
}
