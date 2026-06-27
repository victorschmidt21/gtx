package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const defaultTimeout = 60 * time.Second

// RunResult contém o output e exit code de um comando executado.
type RunResult struct {
	Output   []byte
	ExitCode int
}

// Run executa args como um comando externo, capturando stdout+stderr combinados.
// Herda o ambiente e working directory do processo pai.
// Preserva o exit code original do processo filho.
func Run(args []string, timeout time.Duration) (RunResult, error) {
	if len(args) == 0 {
		return RunResult{}, fmt.Errorf("nenhum comando fornecido")
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir, _ = os.Getwd()
	cmd.Env = os.Environ()

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			err = nil // exit code não-zero não é um erro do runner
		} else if ctx.Err() == context.DeadlineExceeded {
			return RunResult{}, fmt.Errorf("timeout após %s: %s", timeout, args[0])
		} else {
			return RunResult{}, fmt.Errorf("executando %s: %w", args[0], err)
		}
	}

	return RunResult{
		Output:   buf.Bytes(),
		ExitCode: exitCode,
	}, nil
}
