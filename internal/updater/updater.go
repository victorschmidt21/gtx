package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const gtxRepo = "victorschmidt21/gtx"

// overridable in tests
var apiBaseURL = "https://api.github.com"
var downloadBaseURL = "https://github.com"

var platformBinary = map[string]string{
	"windows/amd64": "gtx-windows-amd64.exe",
	"linux/amd64":   "gtx-linux-amd64",
	"darwin/arm64":  "gtx-darwin-arm64",
	"darwin/amd64":  "gtx-darwin-amd64",
}

type releaseResponse struct {
	TagName string `json:"tag_name"`
}

// LatestVersion queries the GitHub Releases API and returns the latest tag name.
func LatestVersion(repo string) (string, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	url := fmt.Sprintf("%s/repos/%s/releases/latest", apiBaseURL, repo)
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var r releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	return r.TagName, nil
}

// PrintVersion prints the current version and indicates if a newer version is available.
// Network errors are silenced — only the local version is printed in that case.
func PrintVersion(current string) {
	fmt.Printf("gtx %s\n", current)
	if current == "dev" {
		return
	}
	latest, err := LatestVersion(gtxRepo)
	if err != nil || latest == "" {
		return
	}
	if latest != current {
		fmt.Printf("Nova versão disponível: %s — execute `gtx update` para atualizar\n", latest)
	}
}

// binaryName returns the release binary filename for the current OS/arch.
// Returns "" if the platform is not supported.
func binaryName() string {
	return platformBinary[runtime.GOOS+"/"+runtime.GOARCH]
}

// SelfUpdate downloads the latest release binary and atomically replaces the current executable.
func SelfUpdate(current string) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("localizando executável: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("resolvendo symlink: %w", err)
	}
	return selfUpdate(current, exePath)
}

func selfUpdate(current, exePath string) error {
	latest, err := LatestVersion(gtxRepo)
	if err != nil {
		return fmt.Errorf("Erro ao verificar versão: sem conexão com api.github.com")
	}
	if current != "dev" && latest == current {
		fmt.Printf("gtx %s já é a versão mais recente.\n", current)
		return nil
	}
	name := binaryName()
	if name == "" {
		return fmt.Errorf("plataforma %s/%s não suportada — compile via `go install github.com/victorschmidt21/gtx/cmd/gtx@latest`", runtime.GOOS, runtime.GOARCH)
	}

	fmt.Printf("Atualizando gtx %s → %s...\n", current, latest)

	url := fmt.Sprintf("%s/victorschmidt21/gtx/releases/download/%s/%s", downloadBaseURL, latest, name)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("baixando binário: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download falhou: HTTP %d", resp.StatusCode)
	}

	dir := filepath.Dir(exePath)
	tmpPath := filepath.Join(dir, "gtx.new"+exeSuffix())

	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		if os.IsPermission(err) {
			return permissionError()
		}
		return fmt.Errorf("criando arquivo temporário: %w", err)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("gravando binário: %w", err)
	}
	f.Close()

	bakPath := exePath + ".old"
	if err := os.Rename(exePath, bakPath); err != nil {
		os.Remove(tmpPath)
		if os.IsPermission(err) {
			return permissionError()
		}
		return fmt.Errorf("renomeando binário atual: %w", err)
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		os.Rename(bakPath, exePath) // restore
		os.Remove(tmpPath)
		return fmt.Errorf("instalando novo binário: %w", err)
	}

	os.Remove(bakPath) // silencioso se falhar (arquivo em uso)

	fmt.Println("Download concluído. GTX atualizado com sucesso!")
	return nil
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func permissionError() error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("sem permissão para atualizar o binário — execute o terminal como Administrador")
	}
	return fmt.Errorf("sem permissão para atualizar o binário — tente `sudo gtx update`")
}
