package filter_test

import (
	"testing"

	"github.com/victorschmidt21/gtx/internal/filter"
)

func TestPipeline_StripANSI(t *testing.T) {
	p := &filter.Pipeline{Name: "test", StripANSI: true}
	res, err := p.Apply([]byte("\x1b[32mgreen text\x1b[0m"), filter.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if string(res.Output) != "green text" {
		t.Errorf("esperava 'green text', obteve: %q", res.Output)
	}
}

func TestPipeline_MaxLines(t *testing.T) {
	p := &filter.Pipeline{Name: "test", MaxLines: 2}
	input := "line1\nline2\nline3\nline4\n"
	res, err := p.Apply([]byte(input), filter.Context{})
	if err != nil {
		t.Fatal(err)
	}
	out := string(res.Output)
	if !contains(out, "omitidas") {
		t.Errorf("esperava indicação de linhas omitidas, obteve: %q", out)
	}
}

func TestPipeline_OnEmpty(t *testing.T) {
	p := &filter.Pipeline{Name: "test", OnEmpty: "ok (vazio)"}
	res, err := p.Apply([]byte(""), filter.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if string(res.Output) != "ok (vazio)" {
		t.Errorf("esperava 'ok (vazio)', obteve: %q", res.Output)
	}
}

func TestPipeline_MutuallyExclusive(t *testing.T) {
	p := &filter.Pipeline{
		Name:               "test",
		StripLinesMatching: []string{"foo"},
		KeepLinesMatching:  []string{"bar"},
	}
	_, err := p.Apply([]byte("some input"), filter.Context{})
	if err == nil {
		t.Fatal("esperava erro de configuração, obteve nil")
	}
}

func TestPipeline_MatchOutputShortCircuit(t *testing.T) {
	p := &filter.Pipeline{
		Name: "test",
		MatchOutput: []filter.MatchRule{
			{Pattern: "up-to-date", Value: "ok (já atualizado)"},
		},
		MaxLines: 1, // não deve ser aplicado pois match faz short-circuit
	}
	input := "Everything up-to-date\nsome other line\n"
	res, err := p.Apply([]byte(input), filter.Context{})
	if err != nil {
		t.Fatal(err)
	}
	if string(res.Output) != "ok (já atualizado)" {
		t.Errorf("esperava short-circuit, obteve: %q", res.Output)
	}
}

func TestEstimateTokens(t *testing.T) {
	cases := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"abcd", 1},
		{"abcde", 2},
		{"hello world", 3},
	}
	for _, c := range cases {
		got := filter.EstimateTokens([]byte(c.input))
		if got != c.expected {
			t.Errorf("EstimateTokens(%q) = %d, esperava %d", c.input, got, c.expected)
		}
	}
}
