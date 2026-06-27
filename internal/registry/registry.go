package registry

import (
	"strings"

	"github.com/victorschmidt21/gtx/internal/filter"
	"github.com/victorschmidt21/gtx/internal/spec"
)

// CommandInfo contém metadados de exibição de um comando.
type CommandInfo struct {
	Key           string
	Description   string
	Reduction     string
	OutputExample string
}

// Entry é o que o registry armazena por chave de comando.
// CmdArgs são args extras injetados no runner (não visíveis ao usuário).
type Entry struct {
	Filter  filter.Filter
	CmdArgs []string
	Info    CommandInfo
}

// Registry mapeia chaves de comando para entradas com filtro + args.
type Registry struct {
	entries      map[string]Entry
	filterByName map[string]filter.Filter
}

// New cria um registry carregando filtros built-in e a spec YAML.
// O YAML é fonte de verdade para cmd_args, metadados e mapeamento de filtros.
func New(specPath string) *Registry {
	r := &Registry{
		entries: make(map[string]Entry),
	}
	r.registerBuiltins()

	s, err := spec.Load(specPath)
	if err == nil {
		r.loadFromSpec(s.Commands, nil)
	}
	return r
}

func key(parts []string) string {
	return strings.Join(parts, " ")
}

// LookupEntry retorna a Entry completa para os args fornecidos.
// Tenta do caminho mais específico ao menos específico (suporte a subcomandos).
func (r *Registry) LookupEntry(parts ...string) (Entry, bool) {
	for i := len(parts); i > 0; i-- {
		k := key(parts[:i])
		if e, ok := r.entries[k]; ok && e.Filter != nil {
			return e, true
		}
	}
	return Entry{}, false
}

// Lookup é atalho para verificar existência de filtro (usado pelo rewrite).
func (r *Registry) Lookup(parts ...string) (filter.Filter, bool) {
	e, ok := r.LookupEntry(parts...)
	return e.Filter, ok
}

// ListCommands retorna metadados de todos os comandos com filtro registrado.
func (r *Registry) ListCommands() []CommandInfo {
	seen := make(map[string]struct{})
	result := make([]CommandInfo, 0)
	for _, e := range r.entries {
		if e.Filter == nil {
			continue
		}
		if _, already := seen[e.Info.Key]; already {
			continue
		}
		seen[e.Info.Key] = struct{}{}
		result = append(result, e.Info)
	}
	return result
}

// loadFromSpec percorre a spec YAML e popula r.entries.
func (r *Registry) loadFromSpec(cmds spec.CommandsMap, prefix []string) {
	for name, cmd := range cmds {
		parts := append(append([]string{}, prefix...), name)
		k := key(parts)

		entry := Entry{
			CmdArgs: cmd.CmdArgs,
			Info: CommandInfo{
				Key:           k,
				Description:   cmd.Description,
				Reduction:     cmd.Reduction,
				OutputExample: cmd.OutputExample,
			},
		}

		if cmd.Filter != "" {
			if f, ok := r.filterByName[cmd.Filter]; ok {
				entry.Filter = f
			}
		}

		r.entries[k] = entry

		if cmd.Subcommands != nil {
			r.loadFromSpec(cmd.Subcommands, parts)
		}
	}
}
