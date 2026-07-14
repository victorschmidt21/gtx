package filter

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// GitStatusFilter comprime `git status` em staged/modified/untracked.
type GitStatusFilter struct{}

func (f *GitStatusFilter) Name() string { return "git_status" }

func (f *GitStatusFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	text := string(output)

	// exit != 0 (ex: fora de um repo) — preserva o erro em vez de "ok (limpo)"
	if ctx.ExitCode != 0 {
		return result(f.Name(), strings.TrimSpace(text), tokensIn), nil
	}

	var staged, modified, deleted, untracked []string

	for _, line := range strings.Split(text, "\n") {
		if len(line) < 3 {
			continue
		}
		xy := line[:2]
		file := strings.TrimSpace(line[3:])
		// Handle renamed files: "old -> new"
		if idx := strings.Index(file, " -> "); idx >= 0 {
			file = file[idx+4:]
		}
		switch {
		case xy[0] == 'M' || xy[0] == 'A' || xy[0] == 'D' || xy[0] == 'R' || xy[0] == 'C':
			staged = append(staged, file)
		case xy[1] == 'M':
			modified = append(modified, file)
		case xy[1] == 'D':
			deleted = append(deleted, file)
		case xy == "??":
			untracked = append(untracked, file)
		}
	}

	if len(staged)+len(modified)+len(deleted)+len(untracked) == 0 {
		return result(f.Name(), "ok (limpo)", tokensIn), nil
	}

	var parts []string
	if len(staged) > 0 {
		parts = append(parts, formatFileList("staged", staged))
	}
	if len(modified) > 0 {
		parts = append(parts, formatFileList("modified", modified))
	}
	if len(deleted) > 0 {
		parts = append(parts, formatFileList("deleted", deleted))
	}
	if len(untracked) > 0 {
		parts = append(parts, formatFileList("untracked", untracked))
	}

	out := strings.Join(parts, "\n")
	return result(f.Name(), out, tokensIn), nil
}

func formatFileList(label string, files []string) string {
	if len(files) <= 5 {
		return fmt.Sprintf("%s: %s", label, strings.Join(files, ", "))
	}
	return fmt.Sprintf("%s: %d arquivos", label, len(files))
}

// GitLogFilter comprime `git log` em one-liners.
type GitLogFilter struct{}

func (f *GitLogFilter) Name() string { return "git_log" }

var logLineRe = regexp.MustCompile(`^([0-9a-f]{7,})\s+(.+)\s+\((.+)\)\s*$`)
var logFormatRe = regexp.MustCompile(`^([0-9a-f]{7,40})\s(.+)$`)

func (f *GitLogFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	text := string(output)

	// git log --oneline já está no formato correto: "abc1234 mensagem"
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		if line == "" {
			continue
		}
		if logFormatRe.MatchString(line) {
			m := logFormatRe.FindStringSubmatch(line)
			hash := m[1]
			if len(hash) > 7 {
				hash = hash[:7]
			}
			lines = append(lines, fmt.Sprintf("%s %s", hash, m[2]))
		} else {
			// formato verboso: extrai hash e subject
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "commit ") {
				// pula — o subject vem depois
				continue
			}
			if line == "" || strings.HasPrefix(line, "Author:") || strings.HasPrefix(line, "Date:") || strings.HasPrefix(line, "Merge:") {
				continue
			}
			// subject line (linha após o commit header)
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 {
		return result(f.Name(), "ok (sem commits)", tokensIn), nil
	}
	return result(f.Name(), strings.Join(lines, "\n"), tokensIn), nil
}

// GitDiffFilter comprime `git diff` mantendo contexto mínimo.
type GitDiffFilter struct{}

func (f *GitDiffFilter) Name() string { return "git_diff" }

func (f *GitDiffFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	text := strings.TrimSpace(string(output))
	if text == "" {
		// --quiet: output vazio com exit != 0 significa que HÁ alterações
		if ctx.ExitCode != 0 {
			return result(f.Name(), "há alterações (exit 1)", tokensIn), nil
		}
		return result(f.Name(), "ok (sem alterações)", tokensIn), nil
	}

	var kept []string
	for _, line := range strings.Split(text, "\n") {
		// mantém: ---/+++ (identificam o arquivo), hunks @@ e linhas +/-.
		// remove: "diff --git" e "index" (redundantes) e linhas de contexto.
		if strings.HasPrefix(line, "@@") ||
			strings.HasPrefix(line, "+") ||
			strings.HasPrefix(line, "-") {
			kept = append(kept, line)
		}
	}

	if len(kept) == 0 {
		return result(f.Name(), "ok (sem alterações)", tokensIn), nil
	}
	return result(f.Name(), strings.Join(kept, "\n"), tokensIn), nil
}

// GitSimpleFilter trata git add → "ok" em sucesso.
type GitSimpleFilter struct{ Verb string }

func (f *GitSimpleFilter) Name() string { return "git_simple_ok" }

func (f *GitSimpleFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	text := strings.TrimSpace(string(output))
	if ctx.ExitCode != 0 {
		if text != "" {
			return result(f.Name(), text, tokensIn), nil
		}
		return result(f.Name(), fmt.Sprintf("erro: %s falhou (exit %d)", f.Verb, ctx.ExitCode), tokensIn), nil
	}
	return result(f.Name(), "ok", tokensIn), nil
}

// GitCommitFilter extrai hash do commit criado.
type GitCommitFilter struct{}

func (f *GitCommitFilter) Name() string { return "git_commit" }

var commitHashRe = regexp.MustCompile(`\[[\w/.-]+\s+([0-9a-f]{7,})\]`)
var nothingToCommitRe = regexp.MustCompile(`nothing to commit|nada para fazer commit|nothing added`)

func (f *GitCommitFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	text := string(output)

	if ctx.ExitCode != 0 || nothingToCommitRe.MatchString(text) {
		return result(f.Name(), "erro: nada para commitar", tokensIn), nil
	}

	if m := commitHashRe.FindStringSubmatch(text); m != nil {
		return result(f.Name(), "ok "+m[1], tokensIn), nil
	}
	return result(f.Name(), "ok", tokensIn), nil
}

// GitPushFilter extrai o branch do push.
type GitPushFilter struct{}

func (f *GitPushFilter) Name() string { return "git_push" }

var upToDateRe = regexp.MustCompile(`Everything up-to-date|Tudo atualizado|already up.to.date`)
var branchRe = regexp.MustCompile(`(?:refs/heads/|-> )(\S+)`)

func (f *GitPushFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	text := string(output)

	if ctx.ExitCode != 0 {
		// preserva linhas de erro relevantes
		var errs []string
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "error:") || strings.HasPrefix(line, "remote: error") || strings.HasPrefix(line, "!") {
				errs = append(errs, line)
			}
		}
		if len(errs) > 0 {
			return result(f.Name(), strings.Join(errs, "\n"), tokensIn), nil
		}
		return result(f.Name(), fmt.Sprintf("erro: push falhou (exit %d)", ctx.ExitCode), tokensIn), nil
	}

	if upToDateRe.MatchString(text) {
		return result(f.Name(), "ok (já atualizado)", tokensIn), nil
	}

	if m := branchRe.FindStringSubmatch(text); m != nil {
		return result(f.Name(), "ok "+m[1], tokensIn), nil
	}
	return result(f.Name(), "ok", tokensIn), nil
}

// GitPullFilter extrai estatísticas do pull.
type GitPullFilter struct{}

func (f *GitPullFilter) Name() string { return "git_pull" }

var pullStatsRe = regexp.MustCompile(`(\d+) files? changed(?:, (\d+) insertions?\(\+\))?(?:, (\d+) deletions?\(-\))?`)
var alreadyUpToDateRe = regexp.MustCompile(`Already up.to.date|Já está atualizado`)

func (f *GitPullFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	text := string(output)

	if ctx.ExitCode != 0 {
		return result(f.Name(), fmt.Sprintf("erro: pull falhou (exit %d)", ctx.ExitCode), tokensIn), nil
	}

	if alreadyUpToDateRe.MatchString(text) {
		return result(f.Name(), "ok (já atualizado)", tokensIn), nil
	}

	if m := pullStatsRe.FindStringSubmatch(text); m != nil {
		files := m[1]
		ins := m[2]
		del := m[3]
		s := fmt.Sprintf("ok %s arquivo(s)", files)
		if ins != "" {
			s += " +" + ins
		}
		if del != "" {
			s += " -" + del
		}
		return result(f.Name(), s, tokensIn), nil
	}
	return result(f.Name(), "ok", tokensIn), nil
}

func result(name, text string, tokensIn int) Result {
	out := []byte(text)
	return Result{
		Output:     out,
		TokensIn:   tokensIn,
		TokensOut:  EstimateTokens(out),
		FilterName: name,
	}
}

// relativeTime formata uma duração em texto legível (não usado externamente,
// mas mantido para uso futuro nos filtros de log).
func relativeTime(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds atrás", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm atrás", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh atrás", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd atrás", int(d.Hours()/24))
	}
}
