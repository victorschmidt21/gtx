package rewrite

import (
	"strings"

	"github.com/victorschmidt21/gtx/internal/registry"
)

// Rewrite recebe uma linha de comando e retorna a versão reescrita.
// Se o comando tem um filtro registrado, prefixa com "gtx".
// Preserva pipes: apenas a parte antes do primeiro "|" é prefixada.
// Sempre retorna exit 0 (o chamador nunca deve propagar erro).
func Rewrite(line string, reg *registry.Registry) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return line
	}

	// separa a parte antes do primeiro pipe
	pipeIdx := indexPipe(line)
	rawCmd := line   // preserva espaçamento original para o output
	pipeSuffix := ""
	if pipeIdx >= 0 {
		rawCmd = line[:pipeIdx]
		pipeSuffix = line[pipeIdx:] // inclui o "|" e espaço
	}

	parts := strings.Fields(rawCmd)
	if len(parts) == 0 {
		return line
	}

	if _, ok := reg.Lookup(parts...); ok {
		return "gtx " + rawCmd + pipeSuffix
	}
	return line
}

// indexPipe encontra a posição do primeiro "|" fora de aspas.
func indexPipe(s string) int {
	inSingle := false
	inDouble := false
	for i, ch := range s {
		switch ch {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '|':
			if !inSingle && !inDouble {
				return i
			}
		}
	}
	return -1
}
