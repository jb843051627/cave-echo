package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// staticHandler serves the embedded console from disk with safe path joins.
func (s *Server) staticHandler() http.Handler {
	fs := http.FileServer(http.Dir(s.static))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		clean := filepath.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
		full := filepath.Join(s.static, clean)
		info, err := os.Stat(full)
		if err != nil || info.IsDir() {
			// fall back to the single-page console entry
			indexPath := filepath.Join(s.static, "index.html")
			if _, indexErr := os.Stat(indexPath); indexErr != nil {
				writeError(w, http.StatusNotFound, "static console not deployed")
				return
			}
			r2 := new(http.Request)
			*r2 = *r
			r2.URL.Path = "/"
			fs.ServeHTTP(w, r2)
			return
		}
		fs.ServeHTTP(w, r)
	})
}
