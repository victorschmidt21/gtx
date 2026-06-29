package rewrite_test

import (
	"testing"

	"github.com/victorschmidt21/gtx/internal/registry"
	"github.com/victorschmidt21/gtx/internal/rewrite"
)

func newReg() *registry.Registry {
	return registry.New("../../spec/commands.yaml")
}

func TestRewrite_WithFilter(t *testing.T) {
	reg := newReg()
	out := rewrite.Rewrite("git status", reg)
	if out != "gtx git status" {
		t.Errorf("esperava 'gtx git status', obteve: %q", out)
	}
}

func TestRewrite_WithoutFilter(t *testing.T) {
	reg := newReg()
	out := rewrite.Rewrite("kubectl apply -f deploy.yaml", reg)
	if out != "kubectl apply -f deploy.yaml" {
		t.Errorf("esperava passthrough, obteve: %q", out)
	}
}

func TestRewrite_NpmInstall(t *testing.T) {
	reg := newReg()
	out := rewrite.Rewrite("npm install", reg)
	if out != "gtx npm install" {
		t.Errorf("esperava 'gtx npm install', obteve: %q", out)
	}
}

func TestRewrite_WithPipe(t *testing.T) {
	reg := newReg()
	out := rewrite.Rewrite("git status | grep modified", reg)
	if out != "gtx git status | grep modified" {
		t.Errorf("esperava prefixação antes do pipe, obteve: %q", out)
	}
}

func TestRewrite_EmptyLine(t *testing.T) {
	reg := newReg()
	out := rewrite.Rewrite("", reg)
	if out != "" {
		t.Errorf("linha vazia deve ser passthrough, obteve: %q", out)
	}
}

func TestRewrite_WithArgs(t *testing.T) {
	reg := newReg()
	out := rewrite.Rewrite("git log -n 20 --oneline", reg)
	if out != "gtx git log -n 20 --oneline" {
		t.Errorf("esperava preservação de args, obteve: %q", out)
	}
}

func TestRewrite_DockerCompose(t *testing.T) {
	reg := newReg()
	out := rewrite.Rewrite("docker compose ps", reg)
	if out != "gtx docker compose ps" {
		t.Errorf("esperava reescrita de subcomando aninhado, obteve: %q", out)
	}
}
