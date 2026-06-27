package filter_test

import (
	"testing"

	"github.com/victorschmidt21/gtx/internal/filter"
)

func TestGhPrListFilter_Empty(t *testing.T) {
	f := &filter.GhPrListFilter{}
	out := applyFilter(t, f, "", 0)
	if !contains(out, "sem PRs") {
		t.Errorf("esperava 'sem PRs', obteve: %q", out)
	}
}

func TestGhPrListFilter_WithPRs(t *testing.T) {
	f := &filter.GhPrListFilter{}
	input := "42\tfix: corrige bug crítico\tfeat/fix\tOPEN\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "#42") {
		t.Errorf("esperava '#42', obteve: %q", out)
	}
	if !contains(out, "open") {
		t.Errorf("esperava 'open', obteve: %q", out)
	}
}

func TestGhIssueListFilter_Empty(t *testing.T) {
	f := &filter.GhIssueListFilter{}
	out := applyFilter(t, f, "", 0)
	if !contains(out, "sem issues") {
		t.Errorf("esperava 'sem issues', obteve: %q", out)
	}
}

func TestGhIssueListFilter_WithIssues(t *testing.T) {
	f := &filter.GhIssueListFilter{}
	input := "99\ttítulo da issue\tOPEN\tbug\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "#99") {
		t.Errorf("esperava '#99', obteve: %q", out)
	}
}

func TestGhRunListFilter_Empty(t *testing.T) {
	f := &filter.GhRunListFilter{}
	out := applyFilter(t, f, "", 0)
	if !contains(out, "sem runs") {
		t.Errorf("esperava 'sem runs', obteve: %q", out)
	}
}

func TestGhRunListFilter_WithRuns(t *testing.T) {
	f := &filter.GhRunListFilter{}
	input := "completed\tCI\tpush\tmain\t2m\t2m30s\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "CI") {
		t.Errorf("esperava 'CI', obteve: %q", out)
	}
}
