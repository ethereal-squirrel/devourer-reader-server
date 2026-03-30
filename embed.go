package server

import "embed"

//go:embed all:client
var ClientFS embed.FS

//go:embed all:plugins
var PluginsFS embed.FS
