package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

type StaticHandler struct {
	frontendDir string
	basePath    string
}

func NewStaticHandler(frontendDir, basePath string) *StaticHandler {
	return &StaticHandler{frontendDir: frontendDir, basePath: basePath}
}

func (h *StaticHandler) Serve(c echo.Context) error {
	// グループプレフィックスを除いた相対パス
	rel := c.Param("*")
	if strings.HasPrefix(rel, "api/") {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "not found"})
	}
	// パストラバーサル対策
	clean := filepath.Join(h.frontendDir, filepath.Clean("/"+rel))
	if info, err := os.Stat(clean); err == nil && !info.IsDir() {
		return c.File(clean)
	}
	return h.serveIndex(c)
}

func (h *StaticHandler) serveIndex(c echo.Context) error {
	raw, err := os.ReadFile(filepath.Join(h.frontendDir, "index.html"))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "not found"})
	}
	html := injectBaseHref(string(raw), h.basePath+"/")
	return c.HTMLBlob(http.StatusOK, []byte(html))
}

func injectBaseHref(html, baseHref string) string {
	const headTag = "<head>"
	i := strings.Index(html, headTag)
	if i == -1 {
		return html
	}
	tag := "\n    <base href=\"" + baseHref + "\" />"
	pos := i + len(headTag)
	return html[:pos] + tag + html[pos:]
}
