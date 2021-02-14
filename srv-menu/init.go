package srvmenu

import "sangjihaksik/share"

const (
	민주학생 = iota
	민주교직
	창조학생
	창조교직
)

func init() {

	share.HttpMux.HandleFunc("/menu", handleHttp)
	share.SkillMux.F("/skill/menu", skillHandler)
}
