package assets

import (
	"net/http"
	"time"

	"gnd.la/internal/httpserve"
	"gnd.la/log"
)

// Handler returns an http.handlerFunc which serves the assets from this
// Manager. To avoid circular imports, this function returns an http.HandlerFunc
// rather than a gnd.la/app.Handler. To obtain a gnd.la/app.Handler use
// gnd.la/app.HandlerFromHTTPFunc.
func (m *Manager) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := m.Path(r.URL)
		f, err := m.Load(p)
		if err != nil {
			log.Warningf("error serving %s: %s", r.URL, err)
			return
		}
		seeker, err := Seeker(f)
		if err != nil {
			log.Warningf("error serving %s: %s", r.URL, err)
			return
		}
		var modtime time.Time
		if st, err := m.VFS().Stat(p); err == nil {
			modtime = st.ModTime()
		}
		if r.URL.RawQuery != "" {
			httpserve.NeverExpires(w)
		}
		http.ServeContent(w, r, r.URL.Path, modtime, seeker)
		f.Close()
	}
}
