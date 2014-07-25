package assets

import (
	"net/http"
	"time"

	"gnd.la/internal/httpserve"
	"gnd.la/log"
)

func Handler(m *Manager) http.HandlerFunc {
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
