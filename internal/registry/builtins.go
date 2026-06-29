package registry

import "github.com/victorschmidt21/gtx/internal/filter"

// registerBuiltins popula r.filterByName mapeando nome da spec → implementação.
// O mapeamento de nome de filtro para caminho de comando fica no YAML.
func (r *Registry) registerBuiltins() {
	r.filterByName = map[string]filter.Filter{
		"git_status":        &filter.GitStatusFilter{},
		"git_log":           &filter.GitLogFilter{},
		"git_diff":          &filter.GitDiffFilter{},
		"git_simple_ok":     &filter.GitSimpleFilter{Verb: "add"},
		"git_commit":        &filter.GitCommitFilter{},
		"git_push":          &filter.GitPushFilter{},
		"git_pull":          &filter.GitPullFilter{},
		"docker_ps":         &filter.DockerPsFilter{},
		"docker_images":     &filter.DockerImagesFilter{},
		"docker_logs":       &filter.DockerLogsFilter{},
		"docker_compose_ps": &filter.DockerComposePsFilter{},
		"gh_pr_list":        &filter.GhPrListFilter{},
		"gh_pr_view":        &filter.GhPrViewFilter{},
		"gh_issue_list":     &filter.GhIssueListFilter{},
		"gh_run_list":       &filter.GhRunListFilter{},
		"npm_install":       &filter.NpmInstallFilter{},
		"pnpm_install":      &filter.PnpmInstallFilter{},
		"yarn_install":      &filter.YarnInstallFilter{},
	}
}
