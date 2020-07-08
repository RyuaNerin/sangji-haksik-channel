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

const (
	_ = iota
	우산초교
	강원정비기술학원
	터미널앞
	터미널맞은편
	원주역
)

const (
	SchoolToTerminal = 1
	TerminalToSchool = 2
	SchoolToStation  = 3
	StationToSchool  = 4
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
