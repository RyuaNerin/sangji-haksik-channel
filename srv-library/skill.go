package srvlibrary

import (
	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

func skillHandler(ctx *skill.Context) {
	var d *data

	if ctx.Payload.Action.Params != nil {
		if keyRaw, ok := ctx.Payload.Action.Params["key"]; ok {
			if key, ok := keyRaw.(string); ok {
				switch key {
				case "0": // 제1열람실
					d = &seat1
				case "1": // 제2열람실
					d = &seat2
				case "2": // 제3열람실A
					d = &seat3a
				case "3": // 제3열람실B
					d = &seat3b
				case "4": // 그룹스터디실(2층)
					d = &seatRoom
				}
			}
		}
	}

	if d == nil {
		ctx.WriteSimpleText("잘못된 요청입니다.")
		return
	}

	d.lock.RLock()
	defer d.lock.RUnlock()

	if len(d.skillText) == 0 {
		ctx.WriteSimpleText("열람실 정보를 얻어오지 못하였습니다.\n\n잠시 후 다시 시도해주세요.")
	} else {
		// https://i.kakao.com/docs/skill-response-format#basiccard
		res := skill.SkillResponse{
			Version: "2.0",
			Template: skill.SkillTemplate{
				Outputs: []skill.Component{
					skill.Component{
						BasicCard: &skill.BasicCard{
							Title:       d.name,
							Description: d.skillText,
						},
					},
				},
				QuickReplies: []skill.QuickReply{
					skill.QuickReply{
						Label:       "1",
						Action:      "message",
						MessageText: "1",
					},
					skill.QuickReply{
						Label:       "2",
						Action:      "message",
						MessageText: "2",
					},
					skill.QuickReply{
						Label:       "3a",
						Action:      "message",
						MessageText: "3a",
					},
					skill.QuickReply{
						Label:       "3b",
						Action:      "message",
						MessageText: "3b",
					},
					skill.QuickReply{
						Label:       "그룹",
						Action:      "message",
						MessageText: "g",
					},
				},
			},
		}

		if d.enabled {
			// 버튼 추가
			res.Template.Outputs[0].BasicCard.Buttons = []skill.Button{
				skill.Button{
					Label:      "좌석 보기",
					Action:     "webLink",
					WebLinkUrl: d.skillHtmlLink,
				},
			}
		}

		ctx.WriteResponse(&res)
	}
}
