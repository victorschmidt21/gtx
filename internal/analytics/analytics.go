package analytics

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// CommandEntry é um registro de comando filtrado.
type CommandEntry struct {
	Timestamp   time.Time
	OriginalCmd string
	GtxCmd      string
	ProjectPath string
	TokensIn    int
	TokensOut   int
	TokensSaved int
	SavingsPct  float64
	ExecMs      int64
}

// DB é o cliente de analytics.
type DB struct {
	db   *sql.DB
	path string
}

// Open abre (ou cria) o banco de analytics no caminho padrão do OS.
func Open() (*DB, error) {
	path, err := defaultPath()
	if err != nil {
		return nil, err
	}
	return OpenAt(path)
}

// OpenAt abre o banco no caminho especificado (útil para testes).
func OpenAt(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("criando diretório de analytics: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("abrindo banco analytics: %w", err)
	}
	a := &DB{db: db, path: path}
	if err := a.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return a, nil
}

func (a *DB) Close() error { return a.db.Close() }

func (a *DB) migrate() error {
	_, err := a.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`)
	if err != nil {
		return err
	}

	var count int
	a.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count)
	if count == 0 {
		_, err = a.db.Exec(`CREATE TABLE IF NOT EXISTS commands (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp    TEXT NOT NULL,
			original_cmd TEXT NOT NULL,
			gtx_cmd      TEXT NOT NULL,
			project_path TEXT NOT NULL,
			tokens_in    INTEGER NOT NULL,
			tokens_out   INTEGER NOT NULL,
			tokens_saved INTEGER NOT NULL,
			savings_pct  REAL NOT NULL,
			exec_ms      INTEGER NOT NULL
		)`)
		if err != nil {
			return fmt.Errorf("migration 1 falhou: %w", err)
		}
		_, err = a.db.Exec(`INSERT INTO schema_migrations (version, applied_at) VALUES (1, ?)`,
			time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			return err
		}
	}
	return nil
}

// Record insere um registro de comando filtrado.
func (a *DB) Record(e CommandEntry) error {
	_, err := a.db.Exec(`INSERT INTO commands
		(timestamp, original_cmd, gtx_cmd, project_path, tokens_in, tokens_out, tokens_saved, savings_pct, exec_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Timestamp.UTC().Format(time.RFC3339),
		e.OriginalCmd, e.GtxCmd, e.ProjectPath,
		e.TokensIn, e.TokensOut, e.TokensSaved,
		e.SavingsPct, e.ExecMs,
	)
	return err
}

// GainSummary contém estatísticas agregadas de savings.
type GainSummary struct {
	TotalCommands int
	TotalSaved    int
	AvgSavingsPct float64
	TopCommands   []CommandStat
}

// CommandStat é o resumo por comando.
type CommandStat struct {
	Command     string
	Count       int
	TotalSaved  int
	AvgSavingsPct float64
}

// Gain retorna o resumo de tokens economizados.
// Se today == true, filtra apenas o dia atual (UTC).
func (a *DB) Gain(today bool) (*GainSummary, error) {
	where := ""
	var args []interface{}
	if today {
		day := time.Now().UTC().Format("2006-01-02")
		where = "WHERE timestamp LIKE ?"
		args = append(args, day+"%")
	}

	var total int
	var saved int
	var avgPct float64
	row := a.db.QueryRow(fmt.Sprintf(
		`SELECT COUNT(*), COALESCE(SUM(tokens_saved),0), COALESCE(AVG(savings_pct),0) FROM commands %s`, where),
		args...)
	if err := row.Scan(&total, &saved, &avgPct); err != nil {
		return nil, err
	}

	if total == 0 {
		return &GainSummary{}, nil
	}

	rows, err := a.db.Query(fmt.Sprintf(
		`SELECT original_cmd, COUNT(*), SUM(tokens_saved), AVG(savings_pct)
		 FROM commands %s
		 GROUP BY original_cmd
		 ORDER BY SUM(tokens_saved) DESC
		 LIMIT 5`, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var top []CommandStat
	for rows.Next() {
		var cs CommandStat
		if err := rows.Scan(&cs.Command, &cs.Count, &cs.TotalSaved, &cs.AvgSavingsPct); err != nil {
			return nil, err
		}
		top = append(top, cs)
	}

	sort.Slice(top, func(i, j int) bool { return top[i].TotalSaved > top[j].TotalSaved })

	return &GainSummary{
		TotalCommands: total,
		TotalSaved:    saved,
		AvgSavingsPct: avgPct,
		TopCommands:   top,
	}, nil
}

// FormatGain formata o GainSummary para exibição.
func FormatGain(s *GainSummary, today bool) string {
	if s.TotalCommands == 0 {
		return "nenhum dado ainda — execute alguns comandos com gtx"
	}

	period := "total"
	if today {
		period = "hoje"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "tokens economizados (%s)\n", period)
	fmt.Fprintf(&sb, "  comandos filtrados: %d\n", s.TotalCommands)
	fmt.Fprintf(&sb, "  tokens economizados: %d\n", s.TotalSaved)
	fmt.Fprintf(&sb, "  redução média: %.0f%%\n", s.AvgSavingsPct)

	if len(s.TopCommands) > 0 {
		fmt.Fprintf(&sb, "\ntop comandos:\n")
		for _, cs := range s.TopCommands {
			fmt.Fprintf(&sb, "  %-30s  %d execuções  %d tokens economizados  (%.0f%%)\n",
				cs.Command, cs.Count, cs.TotalSaved, cs.AvgSavingsPct)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func defaultPath() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA não definido")
		}
		return filepath.Join(appData, "gtx", "analytics.db"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gtx", "analytics.db"), nil
}
