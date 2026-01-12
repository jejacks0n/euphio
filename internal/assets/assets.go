package assets

import (
	"embed"
)

//go:embed *.yml config/*
var FS embed.FS
