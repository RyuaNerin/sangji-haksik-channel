package main

import (
	"net/http"
)

func handleToday(ctx KakaoContext) {
	keyRaw, ok := ctx.rd.Action.Params["key"]
	if !ok {
		ctx.gin.Status(http.StatusBadRequest)
		return
	}

	key, ok := keyRaw.(string)
	if !ok {
		ctx.gin.Status(http.StatusBadRequest)
		return
	}

	menu, index, ok := getMenu()
	if !ok {
		ctx.WriteSimpleText("죄송합니다.\n식단표를 불러오지 못하였습니다.")
	} else {
		switch key {
		case "0": // 민주관 학생
			ctx.WriteSimpleText(menu.MinjuStudent.dailyMenu[index])
		case "1": // 민주관 교직
			ctx.WriteSimpleText(menu.MinjuProfessor.dailyMenu[index])
		case "2": // 창조관 학생
			ctx.WriteSimpleText(menu.ChangjoStudent.dailyMenu[index])
		case "3": // 창조관 교직
			ctx.WriteSimpleText(menu.ChangjoProfessor.dailyMenu[index])
		}
	}
}
