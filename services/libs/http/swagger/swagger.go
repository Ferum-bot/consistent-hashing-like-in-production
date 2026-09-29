package swagger

import (
	"net/http"

	"github.com/swaggest/swgui/v5emb"
)

const (
	UIPath   = "/swagger/"
	SpecPath = UIPath + "openapi.yaml"
)

type Docs struct {
	spec []byte
	ui   http.Handler
}

func New(title string, spec []byte) *Docs {
	return &Docs{spec: spec, ui: v5emb.New(title, SpecPath, UIPath)}
}

func (d *Docs) UI(w http.ResponseWriter, r *http.Request) {
	d.ui.ServeHTTP(w, r)
}

func (d *Docs) Spec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")

	_, _ = w.Write(d.spec)
}

func (d *Docs) RedirectToUI(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, UIPath, http.StatusFound)
}
