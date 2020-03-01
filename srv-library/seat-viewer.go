package srvlibrary

import (
	"net/http"
	"strconv"
)

var (
	notFound = []byte("열람실 정보를 얻어오지 못하였습니다. 잠시 후 다시 시도해주세요.")
)

func handleSeat(w http.ResponseWriter, r *http.Request) {
	var d *data

	switch r.FormValue("key") {
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
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	d.lock.RLock()
	defer d.lock.RUnlock()

	if d.webViewbody == nil {
		header := w.Header()
		header.Set("Content-Type", "text/plain;charset=utf-8")

		w.WriteHeader(http.StatusNotFound)
		w.Write(notFound)
	} else {
		header := w.Header()
		header.Set("Content-Type", "text/html;charset=utf-8")
		header.Set("Content-Length", strconv.Itoa(len(d.webViewbody)))

		w.WriteHeader(http.StatusOK)
		w.Write(d.webViewbody)
	}
}
