package web

import "embed"

//go:embed dist/index.html
var content embed.FS

func GetIndex() ([]byte, error) {
	return content.ReadFile("dist/index.html")
}
