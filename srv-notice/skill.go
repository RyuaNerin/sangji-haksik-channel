package srvnotice

import (
	"net/http"
	"strconv"

	"sangjihaksik/share"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

var (
	baseReplies = []skill.QuickReply{
		{
			Label:   "공지",
			Action:  "block",
			BlockId: "5f04048201c9fc00013d56b3",
		},
		{
			Label:   "일반",
			Action:  "block",
			BlockId: "5f040e213e869f00019d01f6",
		},
		{
			Label:   "학사",
			Action:  "block",
			BlockId: "5f040fbee249a600012e4b5e",
		},
		{
			Label:   "장학",
			Action:  "block",
			BlockId: "5f040fca3210ac0001403e16",
		},
		{
			Label:   "등록",
			Action:  "block",
			BlockId: "5f041007e249a600012e4b74",
		},
		{
			Label:   "학술",
			Action:  "block",
			BlockId: "5f087b2627fbed0001a6e29c",
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
