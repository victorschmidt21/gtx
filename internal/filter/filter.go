package filter

// Context contém metadados do comando sendo filtrado.
type Context struct {
	Args        []string
	OriginalCmd string
	ExitCode    int
}

// Result é o output de um filtro aplicado.
type Result struct {
	Output     []byte
	TokensIn   int
	TokensOut  int
	FilterName string
}

// Filter é a interface implementada por todos os filtros GTX.
type Filter interface {
	Name() string
	Apply(output []byte, ctx Context) (Result, error)
}

// ReplaceRule define uma substituição regex.
type ReplaceRule struct {
	Pattern     string
	Replacement string
}

// MatchRule define um short-circuit: se o output casa com Pattern,
// retorna Value imediatamente sem processar os estágios seguintes.
type MatchRule struct {
	Pattern string
	Value   string
}

// Pipeline implementa o motor de filtragem de 8 estágios.
type Pipeline struct {
	Name                string
	StripANSI           bool
	Replace             []ReplaceRule
	MatchOutput         []MatchRule
	StripLinesMatching  []string
	KeepLinesMatching   []string
	TruncateAt          int
	TailLines           int
	MaxLines            int
	OnEmpty             string
}

// EstimateTokens estima a contagem de tokens usando ~4 chars/token.
func EstimateTokens(text []byte) int {
	return (len(text) + 3) / 4
}
