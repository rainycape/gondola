package assets

import (
	"gnd.la/log"
	"net/http"
)

func Handler(m *Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, modtime, err := m.LoadURL(r.URL)
		if err != nil {
			log.Warningf("Error serving %s: %s", r.URL, err)
			return
		}
		defer f.Close()
		if r.URL.RawQuery != "" {
			w.Header().Set("Expires", "Thu, 31 Dec 2037 23:55:55 GMT")
			w.Header().Set("Cache-Control", "max-age=315360000")
		}
		http.ServeContent(w, r, r.URL.Path, modtime, f)
	}
}
