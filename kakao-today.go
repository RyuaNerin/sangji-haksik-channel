package main

import (
	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

func handleToday(ctx *skill.Context) {
	key := ctx.Payload.Action.Params["key"].(string)

	switch key {
	case "0": // 민주관 학생
		ctx.WriteSimpleText(MinjuStudent.GetMenu())
	case "1": // 민주관 교직
		ctx.WriteSimpleText(MinjuProfessor.GetMenu())
	case "2": // 창조관 학생
		ctx.WriteSimpleText(ChangjoStudent.GetMenu())
	case "3": // 창조관 교직
		ctx.WriteSimpleText(ChangjoProfessor.GetMenu())
	default:
		ctx.WriteSimpleText("잘못된 요청입니다.")
	}
}
