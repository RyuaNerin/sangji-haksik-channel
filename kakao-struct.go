package main

type chatBotRequest struct {
	UserRequest userRequest `json:"userRequest"`
	Bot         bot         `json:"bot"`
	Action      action      `json:"action"`
}

type userRequest struct {
	Timezone  string `json:"timezone"`
	Utterance string `json:"utterance"`
	Lang      string `json:"lang"`
	User      user   `json:"user"`
}

type user struct {
	Id         string       `json:"id"`
	Type       string       `json:"type"`
	Properties userProperty `json:"properties"`
}

type userProperty struct {
	PlusFriendUserKey string `json:"plusfriendUserKey"`
	AppUserId         string `json:"appUserId"`
}

type action struct {
	Id           string                 `json:"id"`
	Name         string                 `json:"name"`
	Params       map[string]interface{} `json:"params"`
	DetailParams map[string]detailParam `json:"detailParams"`
}

type detailParam struct {
	Origin    string `json:"origin"`
	Value     string `json:"value"`
	GroupName string `json:"groupName"`
}

type bot struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type chatBotResponse struct {
	Version  string        `json:"version"`
	Template skillTemplate `json:"template"`
}

type skillTemplate struct {
	Outputs []component `json:"outputs"`
}

type component struct {
	SimpleText *simpleText `json:"simpleText,omitempty"`
	BasicCard  *basicCard  `json:"basicCard,omitempty"`
}

type simpleText struct {
	Text string `json:"text"`
}

type basicCard struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Buttons     []button `json:"buttons"`
}

type button struct {
	Label      string `json:"label"`
	Action     string `json:"action"`
	WebLinkUrl string `json:"webLinkUrl"`
}
