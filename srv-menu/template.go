package srvmenu

import "text/template"

type tmplData struct {
	Date  string
	Where string

	Error bool

	MorningTime string
	MorningMenu string
	LunchTime   string
	LunchMenu   string
	DinerTime   string
	DinerMenu   string
}

/**
2020년 2월 2일 토요일
민주관 학생식당

----------------------
아침 (09:00 ~ 10:00)
북어해장국
공기밥
깍두기
----------------------
점심 (11:00 ~ 14:00)
메뉴없음
----------------------
저녁 (17:00 ~ 18:30)
일품:돈가스카레덮밥/쥬시쿨
백반:돈육바베큐볶음
미역국
계란찜
파래김자반
*/
var tmpl = template.Must(template.New("template").Parse(
	`{{ $.Date }}
{{ $.Where }}

{{ if $.Error }}정보 조회 실패
문의 : admin@ryuar.in
{{ else if and (eq (len $.MorningMenu) 0)
               (eq (len $.LunchMenu  ) 0)
               (eq (len $.DinerTime  ) 0) }}메뉴 없음{{ else }}---------------------
{{ if gt (len $.MorningMenu) 0 }}아침 ({{ $.MorningTime }})
{{ $.MorningMenu }}
{{ else }}아침
메뉴없음
{{ end }}---------------------
{{ if gt (len $.LunchMenu) 0 }}점심 ({{ $.LunchTime }})
{{ $.LunchMenu }}
{{ else }}점심
메뉴없음
{{ end }}---------------------
{{ if gt (len $.DinerMenu) 0 }}저녁 ({{ $.DinerTime }})
{{ $.DinerMenu }}
{{ else }}저녁
메뉴없음
{{ end }}
{{ end }}`,
))
