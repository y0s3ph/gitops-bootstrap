package templates

import "embed"

//go:embed all:bootstrap all:apps all:environments all:docs all:hooks
var FS embed.FS
