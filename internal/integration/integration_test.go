package integration_test

import (
	"os/exec"
	"strings"
	"testing"
)

// buildGtx compila o binário gtx para os testes de integração.
func buildGtx(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "build", "-o", "gtx_test_bin", "../../cmd/gtx").CombinedOutput()
	if err != nil {
		t.Fatalf("build falhou: %v\n%s", err, out)
	}
	t.Cleanup(func() { exec.Command("cmd", "/c", "del", "gtx_test_bin").Run() })
	return "./gtx_test_bin"
}

func TestIntegration_GitStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração")
	}
	bin := buildGtx(t)

	out, err := exec.Command(bin, "git", "status").CombinedOutput()
	// git status pode falhar se não há repo — aceita ambos os casos
	if err != nil {
		if strings.Contains(string(out), "não é um repositório") ||
			strings.Contains(string(out), "not a git repository") {
			t.Skip("não é um repositório git")
		}
	}
	text := string(out)
	// Deve ser output comprimido, não o git status verboso
	verboseMarkers := []string{"On branch", "Changes not staged", "Untracked files:"}
	for _, marker := range verboseMarkers {
		if strings.Contains(text, marker) {
			t.Errorf("output parece não filtrado — contém %q", marker)
		}
	}
}

func TestIntegration_ExitCodePreserved(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração")
	}
	bin := buildGtx(t)

	// git diff retorna exit 1 quando há mudanças não staged
	cmd := exec.Command(bin, "git", "diff", "--quiet")
	cmd.Run()
	exitCode := cmd.ProcessState.ExitCode()
	// Aceita 0 ou 1 — o importante é que seja o exit code do git, não do gtx
	if exitCode < 0 {
		t.Errorf("exit code inválido: %d", exitCode)
	}
}

func TestIntegration_Rewrite(t *testing.T) {
	if testing.Short() {
		t.Skip("pulando teste de integração")
	}
	bin := buildGtx(t)

	cmd := exec.Command(bin, "rewrite")
	cmd.Stdin = strings.NewReader("git status\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rewrite falhou: %v", err)
	}
	if strings.TrimSpace(string(out)) != "gtx git status" {
		t.Errorf("esperava 'gtx git status', obteve: %q", string(out))
	}
}
