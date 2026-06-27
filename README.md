# GTX — Go Token eXpressor

Proxy CLI que comprime outputs de `git`, `docker` e `gh` antes de chegarem ao contexto do LLM, economizando 70–92% dos tokens em cada comando.

[![Build & Test](https://github.com/victorschmidt21/gtx/actions/workflows/build.yml/badge.svg)](https://github.com/victorschmidt21/gtx/actions/workflows/build.yml)

---

## Por que o GTX existe

Ferramentas como `git status` e `docker ps` produzem outputs verbosos que consomem centenas de tokens desnecessários no contexto do LLM. O GTX intercepta esses outputs e os comprime para o essencial antes de chegarem ao Claude Code.

```
git status  →  "staged: foo.go\nmodified: bar.go\nuntracked: 3 arquivos"
git push    →  "ok main"
git commit  →  "ok abc1234"
docker ps   →  "nginx  running  0.0.0.0:80->80/tcp"
```

**Diferencial em relação ao RTK:** o GTX funciona no Windows nativo (PowerShell/CMD) sem WSL. O hook é instalado diretamente no `settings.json` do Claude Code via `gtx init`.

---

## Comandos suportados

| Comando | Redução | Exemplo de output |
|---------|---------|-------------------|
| `git status` | ~80% | `staged: foo.go` |
| `git log` | ~80% | `abc1234 feat: add feature` |
| `git diff` | ~75% | `--- a/foo.go +++ b/foo.go @@ -1,3 +1,4 @@` |
| `git add` | ~90% | `ok` |
| `git commit` | ~92% | `ok abc1234` |
| `git push` | ~92% | `ok main` |
| `git pull` | ~85% | `ok 3 arquivos +10 -2` |
| `docker ps` | ~80% | `nginx  running  0.0.0.0:80->80/tcp` |
| `docker images` | ~80% | `nginx:latest  142MB` |
| `docker logs` | ~70% | `health check ok (repetido 47x)` |
| `docker compose ps` | ~75% | `web  running  0.0.0.0:8080->80/tcp` |
| `gh pr list` | ~75% | `#42  fix: bug  [open]  user (2d)` |
| `gh pr view` | ~70% | `fix: bug` / `[open] user → main` / `checks: 3/3 ok` |
| `gh issue list` | ~75% | `#99  título da issue  [open]  (bug)` |
| `gh run list` | ~70% | `✓ CI  main  (2m)` |

Qualquer comando sem filtro é executado normalmente (passthrough transparente).

---

## Instalação

### Windows

**Opção A — Script PowerShell (recomendado)**

```powershell
iwr -useb https://raw.githubusercontent.com/victorschmidt21/gtx/main/install.ps1 | iex
```

Instala o binário em `%USERPROFILE%\.local\bin` e adiciona ao PATH do usuário automaticamente.

**Opção B — go install**

```powershell
go install github.com/victorschmidt21/gtx/cmd/gtx@latest
```

**Opção C — Binário pré-compilado**

Baixe `gtx-windows-amd64.exe` na [página de releases](https://github.com/victorschmidt21/gtx/releases/latest), renomeie para `gtx.exe` e coloque em uma pasta que esteja no PATH.

---

### Linux

**Opção A — Binário pré-compilado**

```bash
curl -L https://github.com/victorschmidt21/gtx/releases/latest/download/gtx-linux-amd64 -o gtx
chmod +x gtx
sudo mv gtx /usr/local/bin/
```

**Opção B — go install**

```bash
go install github.com/victorschmidt21/gtx/cmd/gtx@latest
```

---

### macOS (Apple Silicon)

**Opção A — Binário pré-compilado**

```bash
curl -L https://github.com/victorschmidt21/gtx/releases/latest/download/gtx-darwin-arm64 -o gtx
chmod +x gtx
sudo mv gtx /usr/local/bin/
```

**Opção B — go install**

```bash
go install github.com/victorschmidt21/gtx/cmd/gtx@latest
```

---

## Configurando o hook no Claude Code

O GTX funciona como hook `PreToolUse` do Claude Code. Quando ativo, ele intercepta automaticamente os comandos Bash antes da execução.

```bash
gtx init            # instala o hook no settings.json do Claude Code
gtx init --verify   # verifica se está instalado
gtx init --uninstall  # remove o hook
```

Localização do `settings.json`:
- **Windows:** `%APPDATA%\Claude\settings.json`
- **Linux / macOS:** `~/.claude/settings.json`

O bloco inserido pelo `gtx init`:

```json
{
  "hooks": {
    "PreToolUse": [{
      "matcher": "Bash",
      "hooks": [{ "type": "command", "command": "gtx rewrite" }]
    }]
  }
}
```

Após instalar, reinicie o Claude Code para que o hook entre em vigor.

---

## Uso

O GTX pode ser usado diretamente ou via hook automático:

```bash
# Uso direto
gtx git status
gtx git log -n 10
gtx docker ps
gtx gh pr list

# Ver todos os comandos com filtro registrado
gtx list

# Ver tokens economizados
gtx gain
gtx gain --today
```

Com o hook ativo, os comandos são interceptados e redirecionados automaticamente — você não precisa digitar `gtx` manualmente.

---

## Como funciona

O GTX aplica um pipeline de 8 estágios sobre o output de cada comando:

```
strip_ansi → replace → match_output → strip/keep_lines
→ truncate → tail_lines → max_lines → on_empty
```

O `gtx rewrite` é o hook stdin→stdout registrado no Claude Code: recebe `git status`, verifica se há filtro, e devolve `gtx git status`. O processo filho executa o comando real com flags internas (ex: `--porcelain` para `git status`) e o filtro comprime o output antes de exibir.

Tokens economizados são registrados localmente em SQLite:
- **Linux / macOS:** `~/.config/gtx/analytics.db`
- **Windows:** `%APPDATA%\gtx\analytics.db`

---

## Build a partir do código-fonte

Requer Go 1.22+.

```bash
git clone https://github.com/victorschmidt21/gtx.git
cd gtx

make build      # compila o binário gtx na pasta atual
make test       # roda os testes
make install    # instala em $GOPATH/bin
make cross-compile  # gera binários para Windows, Linux e macOS em dist/
```
