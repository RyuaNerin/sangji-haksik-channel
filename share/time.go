package share

import "strings"

var (
	TimeFormatKr = strings.NewReplacer(
		"am", "오전",
		"pm", "오후",
		"Sun", "일요일",
		"Mon", "월요일",
		"Tue", "화요일",
		"Wed", "수요일",
		"Thu", "목요일",
		"Fri", "금요일",
		"Sat", "토요일",
	)
)
