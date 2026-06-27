package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// DockerPsFilter comprime `docker ps` para NOME/STATUS/PORTAS.
type DockerPsFilter struct{}

func (f *DockerPsFilter) Name() string { return "docker_ps" }

func (f *DockerPsFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Primeira linha é o header — encontrar posições das colunas
	if len(lines) <= 1 {
		return result(f.Name(), "ok (sem containers rodando)", tokensIn), nil
	}

	header := lines[0]
	nameIdx := strings.Index(header, "NAMES")
	statusIdx := strings.Index(header, "STATUS")
	portsIdx := strings.Index(header, "PORTS")

	if nameIdx < 0 || statusIdx < 0 {
		// fallback: só retorna as linhas sem o header
		rows := lines[1:]
		if len(rows) == 0 {
			return result(f.Name(), "ok (sem containers rodando)", tokensIn), nil
		}
		return result(f.Name(), strings.Join(rows, "\n"), tokensIn), nil
	}

	var rows []string
	for _, line := range lines[1:] {
		if len(line) == 0 {
			continue
		}
		// Ordem das colunas: STATUS < PORTS < NAMES
		name := extractCol(line, nameIdx, len(line))
		status := extractCol(line, statusIdx, portsIdx)
		ports := ""
		if portsIdx >= 0 && portsIdx < len(line) {
			ports = extractCol(line, portsIdx, nameIdx)
		}
		name = strings.TrimSpace(name)
		status = strings.TrimSpace(status)
		ports = strings.TrimSpace(ports)
		if name == "" {
			continue
		}
		row := fmt.Sprintf("%-20s  %-20s  %s", name, status, ports)
		rows = append(rows, strings.TrimRight(row, " "))
	}

	if len(rows) == 0 {
		return result(f.Name(), "ok (sem containers rodando)", tokensIn), nil
	}
	return result(f.Name(), strings.Join(rows, "\n"), tokensIn), nil
}

func extractCol(line string, start, end int) string {
	if start >= len(line) {
		return ""
	}
	if end > len(line) {
		end = len(line)
	}
	if end <= start {
		return strings.TrimSpace(line[start:])
	}
	return strings.TrimSpace(line[start:end])
}

// DockerImagesFilter comprime `docker images` para nome:tag + tamanho.
type DockerImagesFilter struct{}

func (f *DockerImagesFilter) Name() string { return "docker_images" }

func (f *DockerImagesFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) <= 1 {
		return result(f.Name(), "ok (sem imagens)", tokensIn), nil
	}

	header := lines[0]
	repoIdx := strings.Index(header, "REPOSITORY")
	tagIdx := strings.Index(header, "TAG")
	sizeIdx := strings.Index(header, "SIZE")

	var rows []string
	for _, line := range lines[1:] {
		if len(line) == 0 {
			continue
		}
		repo := ""
		tag := ""
		size := ""
		if repoIdx >= 0 && tagIdx >= 0 {
			repo = strings.TrimSpace(extractCol(line, repoIdx, tagIdx))
			if sizeIdx >= 0 {
				tag = strings.TrimSpace(extractCol(line, tagIdx, sizeIdx))
				size = strings.TrimSpace(extractCol(line, sizeIdx, len(line)))
			} else {
				tag = strings.TrimSpace(extractCol(line, tagIdx, len(line)))
			}
		}
		if repo == "" {
			continue
		}
		nameTag := repo
		if tag != "" && tag != "<none>" {
			nameTag = repo + ":" + tag
		}
		rows = append(rows, fmt.Sprintf("%-40s  %s", nameTag, size))
	}

	if len(rows) == 0 {
		return result(f.Name(), "ok (sem imagens)", tokensIn), nil
	}
	return result(f.Name(), strings.Join(rows, "\n"), tokensIn), nil
}

// DockerLogsFilter deduplica linhas repetidas em `docker logs`.
type DockerLogsFilter struct{}

func (f *DockerLogsFilter) Name() string { return "docker_logs" }

const dedupThreshold = 3

func (f *DockerLogsFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return result(f.Name(), "ok (sem logs)", tokensIn), nil
	}

	var out []string
	i := 0
	for i < len(lines) {
		current := lines[i]
		count := 1
		for i+count < len(lines) && lines[i+count] == current {
			count++
		}
		if count >= dedupThreshold {
			out = append(out, current)
			if count > 2 {
				out = append(out, fmt.Sprintf("(repetido %dx)", count-2))
			}
			out = append(out, current)
		} else {
			for j := 0; j < count; j++ {
				out = append(out, current)
			}
		}
		i += count
	}

	return result(f.Name(), strings.Join(out, "\n"), tokensIn), nil
}

// DockerComposePsFilter comprime `docker compose ps`.
type DockerComposePsFilter struct{}

func (f *DockerComposePsFilter) Name() string { return "docker_compose_ps" }

var composeSvcRe = regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)\s+(.*)$`)

func (f *DockerComposePsFilter) Apply(output []byte, ctx Context) (Result, error) {
	tokensIn := EstimateTokens(output)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) <= 1 {
		return result(f.Name(), "ok (sem serviços)", tokensIn), nil
	}

	header := lines[0]
	nameIdx := strings.Index(header, "NAME")
	serviceIdx := strings.Index(header, "SERVICE")
	statusIdx := strings.Index(strings.ToUpper(header), "STATUS")
	portsIdx := strings.Index(strings.ToUpper(header), "PORTS")

	var rows []string
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "---") {
			continue
		}
		var svc, status, ports string
		if serviceIdx >= 0 && statusIdx >= 0 {
			if nameIdx >= 0 {
				svc = strings.TrimSpace(extractCol(line, serviceIdx, statusIdx))
			} else {
				svc = strings.TrimSpace(extractCol(line, 0, statusIdx))
			}
			if portsIdx >= 0 {
				status = strings.TrimSpace(extractCol(line, statusIdx, portsIdx))
				ports = strings.TrimSpace(extractCol(line, portsIdx, len(line)))
			} else {
				status = strings.TrimSpace(extractCol(line, statusIdx, len(line)))
			}
		} else {
			// formato simples (docker compose ps v1)
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				svc = fields[0]
				status = fields[len(fields)-1]
			}
		}
		if svc == "" {
			continue
		}
		row := fmt.Sprintf("%-20s  %-15s  %s", svc, status, ports)
		rows = append(rows, strings.TrimRight(row, " "))
	}

	if len(rows) == 0 {
		return result(f.Name(), "ok (sem serviços)", tokensIn), nil
	}
	return result(f.Name(), strings.Join(rows, "\n"), tokensIn), nil
}
