package filter_test

import (
	"testing"

	"github.com/victorschmidt21/gtx/internal/filter"
)

func applyFilter(t *testing.T, f filter.Filter, input string, exitCode int) string {
	t.Helper()
	res, err := f.Apply([]byte(input), filter.Context{ExitCode: exitCode})
	if err != nil {
		t.Fatalf("Apply erro: %v", err)
	}
	return string(res.Output)
}

func TestGitStatusFilter_WithChanges(t *testing.T) {
	f := &filter.GitStatusFilter{}
	input := "M  foo.go\n?? bar.go\n?? baz.go\n M qux.go\n"
	out := applyFilter(t, f, input, 0)
	if out == "" {
		t.Fatal("output vazio")
	}
	// deve conter staged, untracked ou modified
	found := false
	for _, token := range []string{"staged", "untracked", "modified"} {
		if contains(out, token) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("output inesperado: %q", out)
	}
}

func TestGitStatusFilter_Clean(t *testing.T) {
	f := &filter.GitStatusFilter{}
	out := applyFilter(t, f, "", 0)
	if !contains(out, "limpo") {
		t.Errorf("esperava 'limpo', obteve: %q", out)
	}
}

func TestGitLogFilter(t *testing.T) {
	f := &filter.GitLogFilter{}
	input := "abc1234 feat: add feature\ndef5678 fix: bug fix\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "abc1234") {
		t.Errorf("esperava hash abc1234, obteve: %q", out)
	}
}

func TestGitSimpleFilter_Success(t *testing.T) {
	f := &filter.GitSimpleFilter{Verb: "add"}
	out := applyFilter(t, f, "", 0)
	if out != "ok" {
		t.Errorf("esperava 'ok', obteve: %q", out)
	}
}

func TestGitSimpleFilter_Error(t *testing.T) {
	f := &filter.GitSimpleFilter{Verb: "add"}
	out := applyFilter(t, f, "fatal: pathspec 'x' did not match", 128)
	if !contains(out, "fatal") {
		t.Errorf("esperava mensagem de erro original, obteve: %q", out)
	}
}

func TestGitCommitFilter_Success(t *testing.T) {
	f := &filter.GitCommitFilter{}
	input := "[main abc1234] feat: add feature\n 1 file changed\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "abc1234") {
		t.Errorf("esperava hash abc1234, obteve: %q", out)
	}
}

func TestGitCommitFilter_NothingToCommit(t *testing.T) {
	f := &filter.GitCommitFilter{}
	out := applyFilter(t, f, "nothing to commit", 1)
	if !contains(out, "nada") && !contains(out, "erro") {
		t.Errorf("esperava mensagem de nada para commitar, obteve: %q", out)
	}
}

func TestGitPushFilter_Success(t *testing.T) {
	f := &filter.GitPushFilter{}
	input := "To github.com:user/repo.git\n   abc1234..def5678  main -> main\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "main") {
		t.Errorf("esperava 'main', obteve: %q", out)
	}
}

func TestGitPushFilter_UpToDate(t *testing.T) {
	f := &filter.GitPushFilter{}
	out := applyFilter(t, f, "Everything up-to-date\n", 0)
	if !contains(out, "atualizado") {
		t.Errorf("esperava 'atualizado', obteve: %q", out)
	}
}

func TestGitPullFilter_WithChanges(t *testing.T) {
	f := &filter.GitPullFilter{}
	input := "3 files changed, 45 insertions(+), 12 deletions(-)\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "3") {
		t.Errorf("esperava contagem de arquivos, obteve: %q", out)
	}
}

func TestGitPullFilter_UpToDate(t *testing.T) {
	f := &filter.GitPullFilter{}
	out := applyFilter(t, f, "Already up to date.\n", 0)
	if !contains(out, "atualizado") {
		t.Errorf("esperava 'atualizado', obteve: %q", out)
	}
}

func TestGitDiffFilter_Empty(t *testing.T) {
	f := &filter.GitDiffFilter{}
	out := applyFilter(t, f, "", 0)
	if !contains(out, "sem alterações") {
		t.Errorf("esperava 'sem alterações', obteve: %q", out)
	}
}

func TestGitDiffFilter_WithChanges(t *testing.T) {
	f := &filter.GitDiffFilter{}
	input := "diff --git a/foo.go b/foo.go\nindex abc..def 100644\n--- a/foo.go\n+++ b/foo.go\n@@ -1,3 +1,4 @@\n context\n+added line\n-removed line\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "+added line") {
		t.Errorf("esperava linha adicionada, obteve: %q", out)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
