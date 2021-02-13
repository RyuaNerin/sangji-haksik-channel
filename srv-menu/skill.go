package srvmenu

import (
	"net/http"
	"sangjihaksik/share"
	"strconv"
	"time"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

var (
	baseReplies = []skill.QuickReply{
		{
			Label:   "민/학",
			Action:  "block",
			BlockId: "5e54f02a92690d0001ea1080",
		},
		{
			Label:   "민/교",
			Action:  "block",
			BlockId: "5e54f076ffa74800018cb13c",
		},
		{
			Label:   "창/학",
			Action:  "block",
			BlockId: "5e54f1958192ac0001377fc6",
		},
		{
			Label:   "창/교",
			Action:  "block",
			BlockId: "5e54f19a8192ac0001377fc9",
		},
	}

	responseNoWeekend = share.NewSkillDataWithErrorMessage(
		"주말메뉴는 제공되지 않습니다.",
		baseReplies,
	)

	responseError = share.NewSkillDataWithErrorMessage(
		"식단표 정보를 얻어오지 못하였습니다.\n\n잠시 후 다시 시도해주세요.",
		baseReplies,
	)
)

func skillHandler(ctx *skill.Context) {
	now := time.Now()

	weekday := now.Weekday()
	if weekday == time.Sunday || weekday == time.Saturday {
		responseNoWeekend.Serve(ctx)
		return
	}

	key := 0
	if ctx.Payload.Action.Params != nil {
		if keyStrRaw, ok := ctx.Payload.Action.Params["key"]; ok {
			if keyStr, ok := keyStrRaw.(string); ok {
				key, _ = strconv.Atoi(keyStr)
			}
		}
	}

	n, ok := menu[key]
	if !ok {
		ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
	} else {
		day := int(weekday - time.Monday)

		if !n.menu[day].Serve(ctx) {
			responseError.Serve(ctx)
		}
	}
}
