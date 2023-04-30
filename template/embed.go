package template

import (
	"embed"
)

//go:embed static/svg/*.svg
var icons embed.FS

func icon(name string) string {
	b, _ := icons.ReadFile(name)
	return string(b)
}
