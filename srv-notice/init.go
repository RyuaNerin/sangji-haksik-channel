package srvnotice

import "sangjihaksik/share"

const (
	공지사항 = iota
	일반공지
	학사공지
	장학공지
	등록공지
	학술공지
)

func init() {
	share.SkillMux.F("/skill/notice", skillHandler)
}
