//go:build !development

package hub

import (
	"crypto/tls"
	"io/fs"
	"net/http"
	"strings"

	"github.com/henrygd/beszel/internal/hub/utils"
	"github.com/henrygd/beszel/internal/site"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// startServer sets up the production server for Beszel
func (h *Hub) startServer(se *core.ServeEvent) error {
	disableVulnerableHTTP2Server(se.Server)

	indexFile, _ := fs.ReadFile(site.DistDirFS, "index.html")
	html := modifyIndexHTML(h, indexFile)
	basePath := getPublicAppInfo(h).BASE_PATH
	// set up static asset serving
	staticPaths := [2]string{"/static/", "/assets/"}
	serveStatic := apis.Static(site.DistDirFS, false)
	// get CSP configuration
	csp, cspExists := utils.GetEnv("CSP")
	// add route
	se.Router.GET("/{path...}", func(e *core.RequestEvent) error {
		// serve static assets if path is in staticPaths
		for i := range staticPaths {
			if strings.Contains(e.Request.URL.Path, staticPaths[i]) {
				e.Response.Header().Set("Cache-Control", "public, max-age=2592000")
				return serveStatic(e)
			}
		}
		if cspExists {
			e.Response.Header().Del("X-Frame-Options")
			e.Response.Header().Set("Content-Security-Policy", csp)
		}
		// still serve the app for unknown paths (it renders a 404 page),
		// but with a 404 status so scanners and fail2ban see the miss
		status := http.StatusOK
		if !isAppRoute(e.Request.URL.Path, basePath) {
			status = http.StatusNotFound
		}
		return e.HTML(status, html)
	})
	return nil
}

// disableVulnerableHTTP2Server avoids GO-2026-6612 in the Go 1.27.1 HTTP/2
// server while keeping the Hub's HTTP/1.1 HTTPS endpoint available.
func disableVulnerableHTTP2Server(server *http.Server) {
	if server == nil {
		return
	}
	server.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){}
}
