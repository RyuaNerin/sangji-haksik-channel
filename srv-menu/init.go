package srvmenu

import "sangjihaksik/share"

const (
	민주학생 = 0
	민주교직 = 1
	창조학생 = 2
	창조교직 = 3
)

func init() {

	share.HttpMux.HandleFunc("/menu", handleHttp)
	share.SkillMux.F("/skill/menu", skillHandler)
}
