package srvbus

import (
	"net/http"

	"sangjihaksik/share"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
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

var (
	baseReplies = []skill.QuickReply{
		{
			Label:       "학→터",
			Action:      "message",
			MessageText: "학→터",
		},
		{
			Label:       "터→학",
			Action:      "message",
			MessageText: "터→학",
		},
		{
			Label:       "학→원",
			Action:      "message",
			MessageText: "학→원",
		},
		{
			Label:       "원→학",
			Action:      "message",
			MessageText: "원→학",
		},
	}
)
