package static

import (
	"net/http"

	"sangjihaksik/share"
)

func init() {
	share.HttpMux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/static"))))
}
