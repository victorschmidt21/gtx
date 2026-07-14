package filter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/acarl005/stripansi"
)

var (
	// cobre "added 1 package in 4s" e "added 1 package, and audited 2 packages in 711ms"
	npmSuccessRe  = regexp.MustCompile(`added (\d+) packages?.* in ([\d.]+m?s)`)
	npmUpToDateRe = regexp.MustCompile(`up to date`)

	pnpmPackagesRe = regexp.MustCompile(`Packages: \+(\d+)`)
	pnpmDoneRe     = regexp.MustCompile(`Done in ([\d.]+)s`)
	pnpmUpToDateRe = regexp.MustCompile(`Already up.to.date|up to date`)

	yarnDoneRe    = regexp.MustCompile(`Done in ([\d.]+)`)
	yarnUpToDateRe = regexp.MustCompile(`Already up.to.date`)
)

// nodeInstallResult implements the shared success/failure contract for npm/pnpm/yarn.
func nodeInstallResult(name string, raw []byte, ctx Context, successFn func(string) string, errPrefixes []string) Result {
	tokensIn := EstimateTokens(raw)
	text := stripansi.Strip(string(raw))

	if ctx.ExitCode != 0 {
		var errs []string
		for _, line := range strings.Split(text, "\n") {
			l := strings.TrimSpace(line)
			for _, pfx := range errPrefixes {
				if strings.Contains(l, pfx) {
					errs = append(errs, l)
					break
				}
			}
		}
		if len(errs) > 0 {
			out := []byte(strings.Join(errs, "\n"))
			return Result{Output: out, TokensIn: tokensIn, TokensOut: EstimateTokens(out), FilterName: name}
		}
		msg := fmt.Sprintf("erro: %s falhou (exit %d)", name, ctx.ExitCode)
		out := []byte(msg)
		return Result{Output: out, TokensIn: tokensIn, TokensOut: EstimateTokens(out), FilterName: name}
	}

	summary := successFn(text)
	out := []byte(summary)
	return Result{Output: out, TokensIn: tokensIn, TokensOut: EstimateTokens(out), FilterName: name}
}

// NpmInstallFilter compresses `npm install` output.
type NpmInstallFilter struct{}

func (f *NpmInstallFilter) Name() string { return "npm_install" }

func (f *NpmInstallFilter) Apply(output []byte, ctx Context) (Result, error) {
	return nodeInstallResult(f.Name(), output, ctx, func(text string) string {
		if m := npmSuccessRe.FindStringSubmatch(text); m != nil {
			return fmt.Sprintf("ok (%s pacotes, %s)", m[1], m[2])
		}
		if npmUpToDateRe.MatchString(text) {
			return "ok (sem alterações)"
		}
		return "ok"
	}, []string{"npm ERR!", "ERESOLVE"}), nil
}

// PnpmInstallFilter compresses `pnpm install` output.
type PnpmInstallFilter struct{}

func (f *PnpmInstallFilter) Name() string { return "pnpm_install" }

func (f *PnpmInstallFilter) Apply(output []byte, ctx Context) (Result, error) {
	return nodeInstallResult(f.Name(), output, ctx, func(text string) string {
		if pnpmUpToDateRe.MatchString(text) {
			return "ok (sem alterações)"
		}
		count := ""
		if m := pnpmPackagesRe.FindStringSubmatch(text); m != nil {
			count = m[1]
		}
		elapsed := ""
		if m := pnpmDoneRe.FindStringSubmatch(text); m != nil {
			elapsed = m[1]
		}
		if count != "" && elapsed != "" {
			return fmt.Sprintf("ok (%s pacotes, %ss)", count, elapsed)
		}
		if elapsed != "" {
			return fmt.Sprintf("ok (%ss)", elapsed)
		}
		return "ok"
	}, []string{"ERR_PNPM_", "peer dep"}), nil
}

// YarnInstallFilter compresses `yarn install` output (classic and berry).
type YarnInstallFilter struct{}

func (f *YarnInstallFilter) Name() string { return "yarn_install" }

func (f *YarnInstallFilter) Apply(output []byte, ctx Context) (Result, error) {
	return nodeInstallResult(f.Name(), output, ctx, func(text string) string {
		if yarnUpToDateRe.MatchString(text) {
			return "ok (sem alterações)"
		}
		if m := yarnDoneRe.FindStringSubmatch(text); m != nil {
			return fmt.Sprintf("ok (%ss)", m[1])
		}
		return "ok"
	}, []string{"error", "YN0060", "YN0002"}), nil
}
