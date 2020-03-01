package srvlibrary

import (
	"net/http"

	"sangjihaksik/share"
)

const (
	pathWebView       = "/library/seat"
	pathWebViewStatic = "/library/static/"
)

func init() {
	go updateFunc()

	share.SkillMux.F("/skill/library", skillHandler)

	share.HttpMux.HandleFunc(pathWebView, handleSeat)
	share.HttpMux.Handle(pathWebViewStatic, http.StripPrefix(pathWebViewStatic, http.FileServer(http.Dir("srv-library/template/static"))))
}
