package srvmenu

import (
	"net/http"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
	jsoniter "github.com/json-iterator/go"
)

var (
	baseReplies = []skill.QuickReply{
		{
			Label:       "민/학",
			Action:      "message",
			MessageText: "민/학",
		},
		{
			Label:       "민/교",
			Action:      "message",
			MessageText: "민/교",
		},
		{
			Label:       "창/학",
			Action:      "message",
			MessageText: "창/학",
		},
		{
			Label:       "창/교",
			Action:      "message",
			MessageText: "창/교",
		},
	}

	responseNoWeekend, _ = jsoniter.Marshal(
		&skill.SkillResponse{
			Version: "2.0",
			Template: skill.SkillTemplate{
				Outputs: []skill.Component{
					{
						SimpleText: &skill.SimpleText{
							Text: "주말메뉴는 제공되지 않습니다.",
						},
					},
				},
				QuickReplies: baseReplies,
			},
		},
	)

	responseError, _ = jsoniter.Marshal(
		&skill.SkillResponse{
			Version: "2.0",
			Template: skill.SkillTemplate{
				Outputs: []skill.Component{
					{
						SimpleText: &skill.SimpleText{
							Text: "식단표 정보를 얻어오지 못하였습니다.\n\n잠시 후 다시 시도해주세요.",
						},
					},
				},
				QuickReplies: baseReplies,
			},
		},
	)
)

func skillHandler(ctx *skill.Context) {
	var d *data

	if ctx.Payload.Action.Params != nil {
		if keyRaw, ok := ctx.Payload.Action.Params["key"]; ok {
			if key, ok := keyRaw.(string); ok {
				switch key {
				case "0": // 민주관 학생
					d = &minjuStudent
				case "1": // 민주관 교직
					d = &minjuProfessor
				case "2": // 창조관 학생
					d = &changjoStudent
				case "3": // 창조관 교직
					d = &changjoProfessor
				}
			}
		}
	}

	if d == nil {
		ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
	} else {
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
		ctx.ResponseWriter.Write(d.getSkillResponseBytes())
	}
}
