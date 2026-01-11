package assets

import (
	"embed"
)

//go:embed *.asc *.yml
var FS embed.FS
