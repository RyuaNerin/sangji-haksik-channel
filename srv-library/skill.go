package srvlibrary

import (
	"sangjihaksik/share"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

var (
	responseError = share.NewSkillDataWithErrorMessage(
		"열람실 정보를 얻어오지 못하였습니다.\n\n잠시 후 다시 시도해주세요.",
		nil,
	)
)

func skillHandler(ctx *skill.Context) {
	if !skillData.Serve(ctx) {
		responseError.Serve(ctx)
	}
}
