package spec

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	builtin "github.com/victorschmidt21/gtx/spec"
)

// CommandSpec é a spec declarativa de um único subcomando.
type CommandSpec struct {
	Filter        string        `yaml:"filter"`
	CmdArgs       []string      `yaml:"cmd_args"`
	Reduction     string        `yaml:"reduction"`
	OutputExample string        `yaml:"output_example"`
	Description   string        `yaml:"description"`
	Flags         []FlagSpec    `yaml:"flags"`
	Subcommands   CommandsMap   `yaml:"subcommands"`
}

type FlagSpec struct {
	Name    string `yaml:"name"`
	Default string `yaml:"default"`
}

// CommandsMap é um mapa de nome → CommandSpec.
type CommandsMap map[string]CommandSpec

// Spec é o documento raiz de commands.yaml.
type Spec struct {
	Version  string      `yaml:"version"`
	Commands CommandsMap `yaml:"commands"`
}

// Load lê e valida commands.yaml. Se o arquivo não existe, usa a spec
// embutida no binário — garante filtros ativos em instalações standalone.
func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		data, err = builtin.Builtin, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lendo %s: %w", path, err)
	}

	var s Spec
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("YAML inválido em %s: %w", path, err)
	}
	if s.Commands == nil {
		s.Commands = CommandsMap{}
	}
	return &s, nil
}
