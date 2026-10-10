package proxy

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// Page brand marks (same assets as apps/web/public/brand). Dashboard is dark → white mark.
//
//go:embed static/brand/trim-mark-white.png static/brand/trim-mark-black.png
var brandMarkFS embed.FS

const (
	brandMarkWhitePath = "static/brand/trim-mark-white.png"
	brandMarkBlackPath = "static/brand/trim-mark-black.png"
	brandMarkWhiteURL  = "/brand/trim-mark-white.png"
	brandMarkBlackURL  = "/brand/trim-mark-black.png"
)

func brandMarkHandler() http.Handler {
	sub, err := fs.Sub(brandMarkFS, "static/brand")
	if err != nil {
		return http.NotFoundHandler()
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Stable caching for embedded build artifacts.
		w.Header().Set("Cache-Control", "public, max-age=86400")
		// Strip /brand/ prefix for the file server rooted at static/brand.
		r2 := r.Clone(r.Context())
		r2.URL.Path = strings.TrimPrefix(r.URL.Path, "/brand")
		if r2.URL.Path == "" {
			r2.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r2)
	})
}
