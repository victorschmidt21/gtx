package filter

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/acarl005/stripansi"
)

// Apply executa os 8 estágios da pipeline em ordem.
func (p *Pipeline) Apply(output []byte, ctx Context) (Result, error) {
	if len(p.StripLinesMatching) > 0 && len(p.KeepLinesMatching) > 0 {
		return Result{}, fmt.Errorf("pipeline %q: strip_lines e keep_lines são mutuamente exclusivos", p.Name)
	}

	tokensIn := EstimateTokens(output)
	text := string(output)

	// 1. strip_ansi
	if p.StripANSI {
		text = stripansi.Strip(text)
	}

	// 2. replace
	for _, r := range p.Replace {
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return Result{}, fmt.Errorf("replace pattern inválido %q: %w", r.Pattern, err)
		}
		text = re.ReplaceAllString(text, r.Replacement)
	}

	// 3. match_output (short-circuit)
	for _, m := range p.MatchOutput {
		re, err := regexp.Compile(m.Pattern)
		if err != nil {
			return Result{}, fmt.Errorf("match_output pattern inválido %q: %w", m.Pattern, err)
		}
		if re.MatchString(text) {
			out := []byte(m.Value)
			return Result{
				Output:     out,
				TokensIn:   tokensIn,
				TokensOut:  EstimateTokens(out),
				FilterName: p.Name,
			}, nil
		}
	}

	// 4. strip_lines / 5. keep_lines
	if len(p.StripLinesMatching) > 0 {
		text = filterLines(text, p.StripLinesMatching, false)
	} else if len(p.KeepLinesMatching) > 0 {
		text = filterLines(text, p.KeepLinesMatching, true)
	}

	// 6. truncate
	if p.TruncateAt > 0 && len(text) > p.TruncateAt {
		text = text[:p.TruncateAt] + "\n[...truncado]"
	}

	// 7. tail_lines
	if p.TailLines > 0 {
		lines := splitLines(text)
		if len(lines) > p.TailLines {
			lines = lines[len(lines)-p.TailLines:]
		}
		text = strings.Join(lines, "\n")
	}

	// 8. max_lines
	if p.MaxLines > 0 {
		lines := splitLines(text)
		if len(lines) > p.MaxLines {
			omitted := len(lines) - p.MaxLines
			lines = lines[:p.MaxLines]
			text = strings.Join(lines, "\n") + fmt.Sprintf("\n[...%d linhas omitidas]", omitted)
		}
	}

	text = strings.TrimRight(text, "\n")

	// on_empty
	if p.OnEmpty != "" && strings.TrimSpace(text) == "" {
		text = p.OnEmpty
	}

	out := []byte(text)
	return Result{
		Output:     out,
		TokensIn:   tokensIn,
		TokensOut:  EstimateTokens(out),
		FilterName: p.Name,
	}, nil
}

func filterLines(text string, patterns []string, keep bool) string {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pat := range patterns {
		re, err := regexp.Compile(pat)
		if err == nil {
			compiled = append(compiled, re)
		}
	}

	var buf bytes.Buffer
	for _, line := range splitLines(text) {
		matched := false
		for _, re := range compiled {
			if re.MatchString(line) {
				matched = true
				break
			}
		}
		if (keep && matched) || (!keep && !matched) {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}

func splitLines(text string) []string {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}
