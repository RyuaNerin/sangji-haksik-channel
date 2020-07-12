package srvlibrary

import (
	"net/http"
	"strconv"
)

var (
	notFound = []byte("열람실 정보를 얻어오지 못하였습니다. 잠시 후 다시 시도해주세요.")
)

func handleSeat(w http.ResponseWriter, r *http.Request) {
	key, err := strconv.Atoi(r.FormValue("key"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	d, ok := roomMap[key]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if r.FormValue("type") == "png" {
		d.Image.Serve(w, r)
		return
	}

	if d.webBody == nil {
		header := w.Header()
		header.Set("Content-Type", "text/plain; charset=utf-8")

		w.WriteHeader(http.StatusInternalServerError)
		w.Write(notFound)
	} else {
		if lastModified := r.Header.Get("If-None-Match"); len(lastModified) > 0 {
			if lastModified == d.webETag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		header := w.Header()
		header.Set("Content-Type", "text/html; charset=utf-8")
		header.Set("Content-Length", strconv.Itoa(len(d.webBody)))
		header.Set("ETag", d.webETag)
		header.Set("Cache-Control", "max-age=60")

		w.WriteHeader(http.StatusOK)
		w.Write(d.webBody)
	}
}
