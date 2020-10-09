package srvmenu

import (
	"net/http"
	"strconv"
	"time"
)

func handleHttp(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	weekday := now.Weekday()
	if weekday == time.Sunday || weekday == time.Saturday {
		responseNoWeekend.ServeHttp(w, r)
		return
	}

	key := 0

	if keyStr, ok := r.URL.Query()["key"]; ok && len(keyStr) == 1 {
		key, _ = strconv.Atoi(keyStr[0])
	}

	n, ok := menu[key]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		day := int(weekday - time.Monday)

		if !n.menu[day].ServeHttp(w, r) {
			responseError.ServeHttp(w, r)
		}
	}
}
