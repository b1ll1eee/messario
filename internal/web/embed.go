// Package web serves the built Vue client embedded in the binary.
// `make web` writes the Vite build into ./dist before `go build`.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var dist embed.FS

// Handler serves static assets and falls back to index.html for SPA routes.
// When no build is embedded (dev checkout), it returns a short notice.
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return placeholder()
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return placeholder()
	}
	files := http.FS(sub)
	fileServer := http.FileServer(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p != "" {
			if f, err := files.Open("/" + p); err == nil {
				f.Close()
				if strings.HasPrefix(p, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

func placeholder() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html lang="th"><meta charset="utf-8"><title>Messario</title><body style="font-family:system-ui;padding:40px"><h1>Messario</h1><p>API is running. Build the web client with <code>make web</code> to serve it here.</p></body></html>`))
	})
}
