package srvbus

import (
	"net/http"

	"sangjihaksik/share"
)

const (
	httpPathIn  = "/bus/in"
	httpPathOut = "/bus/out"

	pathBase = "srv-bus/public/static"
	pathIn   = pathBase + "/bus-in.htm"
	pathOut  = pathBase + "/bus-out.htm"
)

func init() {
	share.HttpMux.HandleFunc(httpPathIn, serveFile(pathIn))
	share.HttpMux.HandleFunc(httpPathOut, serveFile(pathOut))

	share.SkillMux.F("/skill/bus", skillHandler)
}

func serveFile(filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filename)
	}
}
