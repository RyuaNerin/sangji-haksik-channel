package srvlibrary

import (
	"net/http"
	"sangjihaksik/share"
	"strconv"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

var (
	baseReplies = []skill.QuickReply{
		{
			Label:       "1",
			Action:      "message",
			MessageText: "1",
		},
		{
			Label:       "2",
			Action:      "message",
			MessageText: "2",
		},
		{
			Label:       "3a",
			Action:      "message",
			MessageText: "3a",
		},
		{
			Label:       "3b",
			Action:      "message",
			MessageText: "3b",
		},
		{
			Label:       "그룹",
			Action:      "message",
			MessageText: "g",
		},
	}

	responseError = share.NewSkillDataWithErrorMessage(
		"열람실 정보를 얻어오지 못하였습니다.\n\n잠시 후 다시 시도해주세요.",
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

	n, ok := seat[key]
	if !ok {
		ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
	} else {
		n.skillData.Serve(ctx)
	}
}
