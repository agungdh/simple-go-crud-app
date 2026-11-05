// Package web embeds the AdminLTE templates and static assets so the app can
// be distributed as a single self-contained binary.
package web

import "embed"

// FS contains the templates and static assets served by the application.
//
// The all: prefix is required so files whose names start with an underscore,
// such as dist/css/skins/_all-skins.min.css, are embedded too.
//
//go:embed all:templates all:static
var FS embed.FS
