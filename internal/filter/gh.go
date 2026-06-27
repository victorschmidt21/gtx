package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// GhPrListFilter comprime `gh pr list`.
type GhPrListFilter struct{}

func (f *GhPrListFilter) Name() string { return "gh_pr_list" }

// gh pr list retorna TSV: number \t title \t branch \t state
var prLineRe = regexp.MustCompile(`^(\d+)\t(.+)\t(.+)\t(\w+)`)

func (f *GhPrListFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	text := strings.TrimSpace(string(output))
	if text == "" {
		return result(f.Name(), "ok (sem PRs abertos)", tokensIn), nil
	}

	var rows []string
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) >= 4 {
			num := strings.TrimSpace(fields[0])
			title := truncStr(strings.TrimSpace(fields[1]), 50)
			state := strings.ToLower(strings.TrimSpace(fields[3]))
			rows = append(rows, fmt.Sprintf("#%-5s  %-50s  [%s]", num, title, state))
		} else {
			// formato não-TSV: tenta regex
			if m := prLineRe.FindStringSubmatch(line); m != nil {
				rows = append(rows, fmt.Sprintf("#%-5s  %-50s  [%s]", m[1], truncStr(m[2], 50), strings.ToLower(m[4])))
			} else {
				rows = append(rows, line)
			}
		}
	}

	if len(rows) == 0 {
		return result(f.Name(), "ok (sem PRs abertos)", tokensIn), nil
	}
	return result(f.Name(), strings.Join(rows, "\n"), tokensIn), nil
}

// GhPrViewFilter comprime `gh pr view`.
type GhPrViewFilter struct{}

func (f *GhPrViewFilter) Name() string { return "gh_pr_view" }

var checksRe = regexp.MustCompile(`(\d+) passing.*?(\d+) failing|(\d+)/(\d+) checks`)

func (f *GhPrViewFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	text := string(output)

	title, state, author, base, head, checksSummary := "", "", "", "", "", ""

	for _, line := range strings.Split(text, "\n") {
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "title:"):
			title = strings.TrimSpace(line[6:])
		case strings.HasPrefix(lower, "state:"):
			state = strings.ToLower(strings.TrimSpace(line[6:]))
		case strings.HasPrefix(lower, "author:"):
			author = strings.TrimSpace(line[7:])
		case strings.HasPrefix(lower, "base branch:"):
			base = strings.TrimSpace(line[12:])
		case strings.HasPrefix(lower, "head branch:"):
			head = strings.TrimSpace(line[12:])
		case strings.Contains(lower, "check") && strings.Contains(lower, "pass"):
			if m := checksRe.FindStringSubmatch(lower); m != nil {
				if m[1] != "" {
					checksSummary = fmt.Sprintf("checks: %s ok, %s falhando", m[1], m[2])
				} else {
					checksSummary = fmt.Sprintf("checks: %s/%s ok", m[3], m[4])
				}
			}
		}
	}

	if title == "" {
		return result(f.Name(), strings.TrimSpace(text), tokensIn), nil
	}

	var parts []string
	parts = append(parts, title)
	if author != "" && head != "" && base != "" {
		parts = append(parts, fmt.Sprintf("[%s] %s — %s → %s", state, author, head, base))
	} else if state != "" {
		parts = append(parts, fmt.Sprintf("[%s]", state))
	}
	if checksSummary != "" {
		parts = append(parts, checksSummary)
	}

	return result(f.Name(), strings.Join(parts, "\n"), tokensIn), nil
}

// GhIssueListFilter comprime `gh issue list`.
type GhIssueListFilter struct{}

func (f *GhIssueListFilter) Name() string { return "gh_issue_list" }

func (f *GhIssueListFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	text := strings.TrimSpace(string(output))
	if text == "" {
		return result(f.Name(), "ok (sem issues abertas)", tokensIn), nil
	}

	var rows []string
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) >= 4 {
			num := strings.TrimSpace(fields[0])
			title := truncStr(strings.TrimSpace(fields[1]), 50)
			state := strings.ToLower(strings.TrimSpace(fields[2]))
			labels := ""
			if len(fields) > 3 {
				labels = strings.TrimSpace(fields[3])
			}
			row := fmt.Sprintf("#%-5s  %-50s  [%s]", num, title, state)
			if labels != "" {
				row += "  (" + labels + ")"
			}
			rows = append(rows, row)
		} else {
			rows = append(rows, line)
		}
	}

	if len(rows) == 0 {
		return result(f.Name(), "ok (sem issues abertas)", tokensIn), nil
	}
	return result(f.Name(), strings.Join(rows, "\n"), tokensIn), nil
}

// GhRunListFilter comprime `gh run list`.
type GhRunListFilter struct{}

func (f *GhRunListFilter) Name() string { return "gh_run_list" }

func (f *GhRunListFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	text := strings.TrimSpace(string(output))
	if text == "" {
		return result(f.Name(), "ok (sem runs)", tokensIn), nil
	}

	var rows []string
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) >= 5 {
			status := strings.ToLower(strings.TrimSpace(fields[0]))
			name := strings.TrimSpace(fields[1])
			branch := strings.TrimSpace(fields[3])
			elapsed := ""
			if len(fields) > 5 {
				elapsed = strings.TrimSpace(fields[5])
			}
			icon := statusIcon(status)
			row := fmt.Sprintf("%s %-25s  %-20s", icon, name, branch)
			if elapsed != "" {
				row += "  (" + elapsed + ")"
			}
			rows = append(rows, row)
		} else {
			rows = append(rows, line)
		}
	}

	if len(rows) == 0 {
		return result(f.Name(), "ok (sem runs)", tokensIn), nil
	}
	return result(f.Name(), strings.Join(rows, "\n"), tokensIn), nil
}

func statusIcon(status string) string {
	switch {
	case status == "completed" || status == "success":
		return "✓"
	case status == "failure" || status == "failed":
		return "✗"
	case status == "in_progress" || status == "queued":
		return "⋯"
	default:
		return "?"
	}
}

func truncStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
