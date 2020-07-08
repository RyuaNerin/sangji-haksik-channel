package srvlibrary

import (
	"net/http"

	"sangjihaksik/share"
)

const (
	pathWebView       = "/library/seat"
	pathWebViewStatic = "/library/static/"

	제1열람실  = 0
	제2열람실  = 1
	제3열람실A = 2
	제3열람실B = 3
	그룹스터디실 = 4
)

func init() {
	share.SkillMux.F("/skill/library", skillHandler)

	share.HttpMux.HandleFunc(pathWebView, handleSeat)
	share.HttpMux.Handle(pathWebViewStatic, http.StripPrefix(pathWebViewStatic, http.FileServer(http.Dir("srv-library/template/static"))))
}
