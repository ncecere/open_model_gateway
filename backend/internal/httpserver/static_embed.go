package httpserver

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	fiberfs "github.com/gofiber/fiber/v2/middleware/filesystem"
)

// uiDist contains the compiled frontend assets built by `bun run build`.
//
//go:embed ui/dist
var uiDist embed.FS

const uiDistRoot = "ui/dist"

func embeddedUI() (fs.FS, error) {
	return fs.Sub(uiDist, uiDistRoot)
}

func mountEmbeddedUI(app *fiber.App) {
	dist, err := embeddedUI()
	if err != nil {
		log.Printf("ui assets not embedded: %v", err)
		return
	}

	app.Use("/", fiberfs.New(fiberfs.Config{
		Root:         http.FS(dist),
		PathPrefix:   "",
		Index:        "index.html",
		NotFoundFile: "index.html",
		Browse:       false,
	}))
}

func mountAdminUISubpath(app *fiber.App) {
	dist, err := embeddedUI()
	if err != nil {
		return
	}

	app.Use("/admin/ui", fiberfs.New(fiberfs.Config{
		Root:         http.FS(dist),
		PathPrefix:   "/admin/ui",
		Index:        "index.html",
		NotFoundFile: "index.html",
		Browse:       false,
	}))
}
