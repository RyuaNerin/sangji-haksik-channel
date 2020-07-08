package share

import (
	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

func NewSkillDataWithErrorMessage(str string, baseReplies []skill.QuickReply) *SkillData {
	v := skill.SkillResponse{
		Version: "2.0",
		Template: skill.SkillTemplate{
			Outputs: []skill.Component{
				{
					SimpleText: &skill.SimpleText{
						Text: str,
					},
				},
			},
			QuickReplies: baseReplies,
		},
	}

	d := new(SkillData)
	err := d.Update(&v)
	if err != nil {
		panic(err)
	}

	return d
}
