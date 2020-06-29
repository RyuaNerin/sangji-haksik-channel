package share

import (
	"net/http"
	"runtime"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

var (
	HttpMux  *http.ServeMux = http.NewServeMux()
	SkillMux *skill.MuxHelper
)

func init() {
	HttpMux.HandleFunc(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://pf.kakao.com/_xbkxdyT", http.StatusFound)
		},
	)

	if runtime.GOOS == "windows" {
		SkillMux = skill.NewMuxHelper(HttpMux, "")
	} else {
		SkillMux = skill.NewMuxHelper(HttpMux, "X-Real-IP")
	}
}
