package srvbus

import (
	"net/http"
	"sangjihaksik/share"
	"strconv"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

var (
	stationList = map[int]*stationInfo{
		우산초교: {
			StationName: "우산초교 (정문)",
			RequestBody: []byte("station_id=251061041"),
		},
		강원정비기술학원: {
			StationName: "강원정비기술학원 (상지마트)",
			RequestBody: []byte("station_id=251061013"),
		},
		터미널앞: {
			StationName: "터미널 앞",
			RequestBody: []byte("station_id=251060037"),
		},
		터미널맞은편: {
			StationName: "터미널 길건너",
			RequestBody: []byte("station_id=251060036"),
		},
		원주역: {
			StationName: "원주역 (CU 앞)",
			RequestBody: []byte("station_id=251058010"),
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
