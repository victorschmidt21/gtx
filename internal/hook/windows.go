package hook

import (
	"fmt"
	"os"
	"path/filepath"
)

type windowsHook struct{}

func (h *windowsHook) SettingsPath() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return "", fmt.Errorf("variável APPDATA não encontrada")
	}
	return filepath.Join(appData, "Claude", "settings.json"), nil
}

func (h *windowsHook) Install() error {
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

func (h *windowsHook) Uninstall() error {
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

func (h *windowsHook) Verify() (bool, error) {
	path, err := h.SettingsPath()
	if err != nil {
		return false, err
	}
	return verifyHook(path)
}
