package srvbus

import (
	"net/http"
	"strconv"

	"sangjihaksik/share"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

var (
	baseReplies = []skill.QuickReply{
		{
			Label:   "터→학",
			Action:  "block",
			BlockId: "5efb47942ca48c0001202d84",
		},
		{
			Label:   "학→터",
			Action:  "block",
			BlockId: "5efb4682501c670001e56f7b",
		},
		{
			Label:   "원→학",
			Action:  "block",
			BlockId: "5efb4865031ba400011781cd",
		},
		{
			Label:   "학→원",
			Action:  "block",
			BlockId: "5efb4861031ba400011781c8",
		},
	}

	responseError = share.NewSkillDataWithErrorMessage(
		"실시간 버스 정보를 얻어오지 못하였습니다.\n\n잠시 후 다시 시도해주세요.",
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

	n, ok := routeList[key]
	if !ok {
		ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
	} else {
		if !n.skillData.Serve(ctx) {
			responseError.Serve(ctx)
		}
	}
}
