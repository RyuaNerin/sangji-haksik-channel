package srvnotice

import "sangjihaksik/share"

const (
	공지사항 = 1
	일반공지 = 2
	학사공지 = 3
	장학공지 = 4
	등록공지 = 5
	학술공지 = 6
)

func init() {
	share.SkillMux.F("/skill/notice", skillHandler)
}
