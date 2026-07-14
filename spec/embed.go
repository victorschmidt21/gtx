// Package spec embute o commands.yaml no binário para que o gtx
// funcione standalone (instalado via install.ps1, sem o repositório).
package spec

import _ "embed"

//go:embed commands.yaml
var Builtin []byte
