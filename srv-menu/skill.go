package srvmenu

import (
	skill "github.com/RyuaNerin/go-kakaoskill/v2"
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
		ctx.WriteSimpleText("잘못된 요청입니다.")
	} else {
		res := skill.SkillResponse{
			Version: "2.0",
			Template: skill.SkillTemplate{
				Outputs: []skill.Component{
					skill.Component{
						SimpleText: &skill.SimpleText{
							Text: d.getMenu(),
						},
					},
				},
				QuickReplies: []skill.QuickReply{
					skill.QuickReply{
						Label:       "민/학",
						Action:      "message",
						MessageText: "민/학",
					},
					skill.QuickReply{
						Label:       "민/교",
						Action:      "message",
						MessageText: "민/교",
					},
					skill.QuickReply{
						Label:       "창/학",
						Action:      "message",
						MessageText: "창/학",
					},
					skill.QuickReply{
						Label:       "창/교",
						Action:      "message",
						MessageText: "창/교",
					},
				},
			},
		}

		ctx.WriteResponse(&res)
	}
}
