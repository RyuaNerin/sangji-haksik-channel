package srvlibrary

import (
	"html/template"
	"time"

	"sangjihaksik/share"

	"github.com/getsentry/sentry-go"
)

var tg = template.Must(template.ParseGlob("srv-library/template/*.tmpl.htm"))

const (
	seatStatus = iota
	seatStatusPossible
	seatStatusMan
	seatStatusWoman
	seatStatusFixed
	seatStatusDisabled
)

type templateData struct {
	Name      string // 열람실 이름
	UpdatedAt string // 업데이트 기준일

	DisabledMessage string // 에러용 메시지

	Seat map[int]templateDataSeat
}
type templateDataSeat struct {
	SeatNum string
	Using   bool
}

func (d *data) makeTemplateError(now time.Time, message string) {
	d.webViewBuffer.Reset()

	td := templateData{
		Name:            d.name,
		UpdatedAt:       share.TimeFormatKr.Replace(now.Format("2006년 1월 2일 Mon pm 3시 4분 기준")),
		DisabledMessage: message,
	}

	err := tg.ExecuteTemplate(&d.webViewBuffer, "disabled.tmpl.htm", td)
	if err != nil {
		sentry.CaptureException(err)

		d.webViewbody = nil
		return
	}

	d.webViewbody = d.webViewBuffer.Bytes()
}

func (d *data) makeTemplate(now time.Time) {
	d.webViewBuffer.Reset()

	td := templateData{
		Name:      d.name,
		UpdatedAt: share.TimeFormatKr.Replace(now.Format("2006년 1월 2일 Mon pm 3시 4분 기준")),
		Seat:      d.updateMapBuffer,
	}

	err := tg.ExecuteTemplate(&d.webViewBuffer, d.templateFileName, td)
	if err != nil {
		sentry.CaptureException(err)

		d.webViewbody = nil
		return
	}

	d.webViewbody = d.webViewBuffer.Bytes()
}
