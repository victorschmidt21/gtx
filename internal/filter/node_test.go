package filter_test

import (
	"testing"

	"github.com/victorschmidt21/gtx/internal/filter"
)

func TestNpmInstallFilter_Success(t *testing.T) {
	f := &filter.NpmInstallFilter{}
	input := "\nadded 342 packages in 4.2s\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "342") || !contains(out, "4.2") {
		t.Errorf("esperava contagem e tempo, obteve: %q", out)
	}
}

func TestNpmInstallFilter_UpToDate(t *testing.T) {
	f := &filter.NpmInstallFilter{}
	out := applyFilter(t, f, "up to date, audited 342 packages in 1.2s\n", 0)
	if !contains(out, "sem alterações") {
		t.Errorf("esperava 'sem alterações', obteve: %q", out)
	}
}

func TestNpmInstallFilter_FailureWithANSI(t *testing.T) {
	f := &filter.NpmInstallFilter{}
	input := "\x1b[31mnpm ERR! code ERESOLVE\x1b[0m\nnpm ERR! peer dep conflict\n"
	out := applyFilter(t, f, input, 1)
	if !contains(out, "npm ERR!") {
		t.Errorf("esperava linha de erro, obteve: %q", out)
	}
	if contains(out, "\x1b[") {
		t.Errorf("output não deveria conter ANSI, obteve: %q", out)
	}
}

func TestPnpmInstallFilter_Success(t *testing.T) {
	f := &filter.PnpmInstallFilter{}
	input := "Packages: +342\nDone in 4.2s\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "342") || !contains(out, "4.2") {
		t.Errorf("esperava contagem e tempo, obteve: %q", out)
	}
}

func TestPnpmInstallFilter_Failure(t *testing.T) {
	f := &filter.PnpmInstallFilter{}
	input := "ERR_PNPM_PEER_DEP_ISSUES  Unmet peer dependencies\n"
	out := applyFilter(t, f, input, 1)
	if !contains(out, "ERR_PNPM_") {
		t.Errorf("esperava linha de erro pnpm, obteve: %q", out)
	}
}

func TestYarnInstallFilter_Success(t *testing.T) {
	f := &filter.YarnInstallFilter{}
	input := "[1/4] Resolving packages...\n[4/4] Building fresh packages...\nDone in 4.20s.\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "4.20") {
		t.Errorf("esperava tempo, obteve: %q", out)
	}
}

func TestYarnInstallFilter_Failure(t *testing.T) {
	f := &filter.YarnInstallFilter{}
	input := "error An unexpected error occurred: \"ENOENT\"\n"
	out := applyFilter(t, f, input, 1)
	if !contains(out, "error") {
		t.Errorf("esperava linha de erro yarn, obteve: %q", out)
	}
}
