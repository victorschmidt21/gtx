package filter_test

import (
	"strings"
	"testing"

	"github.com/victorschmidt21/gtx/internal/filter"
)

func TestDockerPsFilter_Empty(t *testing.T) {
	f := &filter.DockerPsFilter{}
	out := applyFilter(t, f, "CONTAINER ID   IMAGE   COMMAND   CREATED   STATUS   PORTS   NAMES\n", 0)
	if !contains(out, "sem containers") {
		t.Errorf("esperava 'sem containers', obteve: %q", out)
	}
}

func TestDockerPsFilter_WithContainers(t *testing.T) {
	f := &filter.DockerPsFilter{}
	input := "CONTAINER ID   IMAGE          COMMAND                  CREATED        STATUS        PORTS                  NAMES\n"
	input += "abc123         nginx:latest   \"/docker-entrypoint…\"   2 hours ago    Up 2 hours    0.0.0.0:80->80/tcp     web\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "web") {
		t.Errorf("esperava nome 'web', obteve: %q", out)
	}
}

func TestDockerImagesFilter_Empty(t *testing.T) {
	f := &filter.DockerImagesFilter{}
	out := applyFilter(t, f, "REPOSITORY   TAG   IMAGE ID   CREATED   SIZE\n", 0)
	if !contains(out, "sem imagens") {
		t.Errorf("esperava 'sem imagens', obteve: %q", out)
	}
}

func TestDockerImagesFilter_WithImages(t *testing.T) {
	f := &filter.DockerImagesFilter{}
	input := "REPOSITORY   TAG       IMAGE ID       CREATED       SIZE\n"
	input += "nginx        latest    abc123def456   2 days ago    142MB\n"
	out := applyFilter(t, f, input, 0)
	if !contains(out, "nginx") {
		t.Errorf("esperava 'nginx', obteve: %q", out)
	}
	if !contains(out, "142MB") {
		t.Errorf("esperava '142MB', obteve: %q", out)
	}
}

func TestDockerLogsFilter_Deduplicate(t *testing.T) {
	f := &filter.DockerLogsFilter{}
	lines := []string{
		"health check ok",
		"health check ok",
		"health check ok",
		"health check ok",
		"health check ok",
		"different line",
	}
	input := strings.Join(lines, "\n")
	out := applyFilter(t, f, input, 0)
	if !contains(out, "repetido") {
		t.Errorf("esperava 'repetido', obteve: %q", out)
	}
	if !contains(out, "different line") {
		t.Errorf("esperava 'different line', obteve: %q", out)
	}
}

func TestDockerLogsFilter_NoDedupBelowThreshold(t *testing.T) {
	f := &filter.DockerLogsFilter{}
	input := "line a\nline a\nline b\n"
	out := applyFilter(t, f, input, 0)
	// 2 repetições está abaixo do threshold de 3 — não deve deduplicar
	if contains(out, "repetido") {
		t.Errorf("não deve deduplicar abaixo do threshold, obteve: %q", out)
	}
}
