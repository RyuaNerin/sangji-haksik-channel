package srvmenu

import "sangjihaksik/share"

func init() {
	go updateFunc()

	share.SkillMux.F("/skill/menu", skillHandler)
}
