package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/victorschmidt21/gtx/internal/analytics"
	"github.com/victorschmidt21/gtx/internal/filter"
	"github.com/victorschmidt21/gtx/internal/hook"
	"github.com/victorschmidt21/gtx/internal/registry"
	"github.com/victorschmidt21/gtx/internal/rewrite"
	"github.com/victorschmidt21/gtx/internal/runner"
)

var version = "dev"

func main() {
	specPath := specFilePath()
	reg := registry.New(specPath)

	root := &cobra.Command{
		Use:     "gtx",
		Short:   "Go Token eXpressor — comprime outputs de comandos para LLMs",
		Version: version,
	}

	// gtx <comando> [args...] — executa comando com filtro
	root.RunE = func(cmd *cobra.Command, args []string) error {
		return runFiltered(reg, args)
	}

	// Subcomando: gtx rewrite
	rewriteCmd := &cobra.Command{
		Use:   "rewrite",
		Short: "Reescreve um comando do stdin para uso como hook PreToolUse",
		RunE: func(cmd *cobra.Command, args []string) error {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Println(rewrite.Rewrite(line, reg))
			}
			return nil
		},
	}

	// Subcomando: gtx init
	var uninstall, verify bool
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Instala o hook GTX no Claude Code settings.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			installer := hook.New()
			switch {
			case uninstall:
				return installer.Uninstall()
			case verify:
				ok, err := installer.Verify()
				if err != nil {
					return err
				}
				path, _ := installer.SettingsPath()
				if ok {
					fmt.Printf("ok (hook instalado em %s)\n", path)
				} else {
					fmt.Printf("hook não instalado — execute `gtx init` para instalar\n")
				}
				return nil
			default:
				return installer.Install()
			}
		},
	}
	initCmd.Flags().BoolVar(&uninstall, "uninstall", false, "Remove o hook GTX")
	initCmd.Flags().BoolVar(&verify, "verify", false, "Verifica se o hook está instalado")

	// Subcomando: gtx gain
	var today bool
	gainCmd := &cobra.Command{
		Use:   "gain",
		Short: "Exibe tokens economizados pelo GTX",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := analytics.Open()
			if err != nil {
				return fmt.Errorf("abrindo banco de analytics: %w", err)
			}
			defer db.Close()

			summary, err := db.Gain(today)
			if err != nil {
				return err
			}
			fmt.Println(analytics.FormatGain(summary, today))
			return nil
		},
	}
	gainCmd.Flags().BoolVar(&today, "today", false, "Mostra apenas os savings de hoje")

	// Subcomando: gtx list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Lista todos os comandos com filtro registrado",
		Run: func(cmd *cobra.Command, args []string) {
			infos := reg.ListCommands()
			if len(infos) == 0 {
				fmt.Println("nenhum comando registrado")
				return
			}
			fmt.Printf("%-30s  %s\n", "COMANDO", "REDUÇÃO")
			for _, info := range infos {
				fmt.Printf("%-30s  %s\n", info.Key, info.Reduction)
			}
		},
	}

	root.AddCommand(rewriteCmd, initCmd, gainCmd, listCmd)

	// ArbitraryArgs: permite `gtx git status`, `gtx git log -n 10`, etc.
	// FParseErrWhitelist: ignora flags desconhecidas (como -n, --oneline)
	// passadas para comandos externos.
	root.Args = cobra.ArbitraryArgs
	root.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func runFiltered(reg *registry.Registry, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("uso: gtx <comando> [args...]")
	}

	entry, hasFilter := reg.LookupEntry(args...)

	// runArgs são os args reais passados ao runner.
	// Se a entry tem CmdArgs, eles são injetados após o subcomando base.
	// Ex: ["git","status"] + CmdArgs["--porcelain"] → ["git","status","--porcelain"]
	// O usuário continua vendo "git status" no output e no analytics.
	runArgs := args
	if hasFilter && len(entry.CmdArgs) > 0 {
		runArgs = append(append([]string{}, args...), entry.CmdArgs...)
	}

	start := time.Now()
	res, err := runner.Run(runArgs, 0)
	execMs := time.Since(start).Milliseconds()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gtx: %v\n", err)
		os.Exit(1)
	}

	ctx := filter.Context{
		Args:        args,
		OriginalCmd: strings.Join(args, " "),
		ExitCode:    res.ExitCode,
	}

	if !hasFilter {
		// passthrough transparente
		os.Stdout.Write(res.Output)
		os.Exit(res.ExitCode)
	}

	filtered, err := entry.Filter.Apply(res.Output, ctx)
	if err != nil {
		// fallback para output original em caso de erro no filtro
		os.Stdout.Write(res.Output)
		os.Exit(res.ExitCode)
	}

	os.Stdout.Write(filtered.Output)
	if len(filtered.Output) > 0 && filtered.Output[len(filtered.Output)-1] != '\n' {
		fmt.Println()
	}

	// analytics síncronos: deve rodar antes de os.Exit.
	// Falhas de analytics são silenciosas — nunca bloqueiam o comando.
	recordAnalytics(ctx, filtered, execMs)

	os.Exit(res.ExitCode)
	return nil
}

func recordAnalytics(ctx filter.Context, filtered filter.Result, execMs int64) {
	db, err := analytics.Open()
	if err != nil {
		return
	}
	defer db.Close()
	cwd, _ := os.Getwd()
	saved := filtered.TokensIn - filtered.TokensOut
	pct := 0.0
	if filtered.TokensIn > 0 {
		pct = float64(saved) / float64(filtered.TokensIn) * 100
	}
	db.Record(analytics.CommandEntry{ //nolint:errcheck
		Timestamp:   time.Now(),
		OriginalCmd: ctx.OriginalCmd,
		GtxCmd:      "gtx " + ctx.OriginalCmd,
		ProjectPath: cwd,
		TokensIn:    filtered.TokensIn,
		TokensOut:   filtered.TokensOut,
		TokensSaved: saved,
		SavingsPct:  pct,
		ExecMs:      execMs,
	})
}

func specFilePath() string {
	// tenta localizar spec/commands.yaml relativo ao executável ou cwd
	candidates := []string{
		"spec/commands.yaml",
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "spec", "commands.yaml"))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "spec/commands.yaml"
}
