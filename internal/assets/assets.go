package assets

import (
	"embed"
)

//go:embed *.asc *.yml art/*.ans
var FS embed.FS
