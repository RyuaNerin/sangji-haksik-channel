package srvnotice

import (
	"net/http"
	"sangjihaksik/share"
	"strconv"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

var (
	baseReplies = []skill.QuickReply{
		{
			Label:       "공지",
			Action:      "message",
			MessageText: "공지사항",
		},
		{
			Label:       "일반",
			Action:      "message",
			MessageText: "일반공지",
		},
		{
			Label:       "학사",
			Action:      "message",
			MessageText: "학사공지",
		},
		{
			Label:       "장학",
			Action:      "message",
			MessageText: "장학공지",
		},
		{
			Label:       "등록",
			Action:      "message",
			MessageText: "등록공지",
		},
	}

	responseError = share.NewSkillDataWithErrorMessage(
		"공지사항을 얻어오지 못하였습니다.\n\n잠시 후 다시 시도해주세요.",
		baseReplies,
	)
)

func skillHandler(ctx *skill.Context) {
	key := 0
	if ctx.Payload.Action.Params != nil {
		if keyStrRaw, ok := ctx.Payload.Action.Params["key"]; ok {
			if keyStr, ok := keyStrRaw.(string); ok {
				key, _ = strconv.Atoi(keyStr)
			}
		}
	}

	n, ok := notice[key]
	if !ok {
		ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
	} else {
		if !n.skillData.Serve(ctx) {
			responseError.Serve(ctx)
		}
	}
}
