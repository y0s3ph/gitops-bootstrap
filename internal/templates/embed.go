package templates

import "embed"

//go:embed all:bootstrap all:apps all:environments all:docs
var FS embed.FS
