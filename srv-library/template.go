package srvlibrary

import (
	"html/template"
	"net/http"
	"time"

	"sangjihaksik/share"

	"github.com/getsentry/sentry-go"
)

const (
	templateDataFormat = "2006년 1월 2일 Mon pm 3시 4분 기준"
)

var tg = template.Must(template.ParseGlob("srv-library/template/*.tmpl.htm"))

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

func (d *data) updateETag(now time.Time) {
	d.webLastModifiedBuf = now.UTC().AppendFormat(d.webLastModifiedBuf[:0], http.TimeFormat)
	d.webLastModified = share.ToString(d.webLastModifiedBuf)
}

func (d *data) makeTemplateError(now time.Time, message string) {
	d.updateETag(now)

	td := templateData{
		Name:            d.name,
		UpdatedAt:       share.TimeFormatKr.Replace(now.Format(templateDataFormat)),
		DisabledMessage: message,
	}

	d.webBodyBuffer.Reset()
	err := tg.ExecuteTemplate(&d.webBodyBuffer, "disabled.tmpl.htm", td)
	if err != nil {
		sentry.CaptureException(err)

		d.webBody = nil
		return
	}

	d.webBody = d.webBodyBuffer.Bytes()
}

func (d *data) makeTemplate(now time.Time) {
	d.updateETag(now)

	td := templateData{
		Name:      d.name,
		UpdatedAt: share.TimeFormatKr.Replace(now.Format(templateDataFormat)),
		Seat:      d.updateMapBuffer,
	}

	d.webBodyBuffer.Reset()
	err := tg.ExecuteTemplate(&d.webBodyBuffer, d.templateFileName, td)
	if err != nil {
		sentry.CaptureException(err)

		d.webBody = nil
		return
	}

	d.webBody = d.webBodyBuffer.Bytes()
}
