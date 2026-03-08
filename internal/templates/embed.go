package templates

import "embed"

//go:embed all:bootstrap all:apps all:environments
var FS embed.FS
