package srvnotice

import (
	"net/http"
	"strconv"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

func skillHandler(ctx *skill.Context) {
	key := 0
	if ctx.Payload.Action.Params != nil {
		if keyStrRaw, ok := ctx.Payload.Action.Params["key"]; ok {
			if keyStr, ok := keyStrRaw.(string); ok {
				key, _ = strconv.Atoi(keyStr)
			}
		}
	}

	n, ok := notice[key]
	if !ok {
		ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
	} else {
		n.skillResponseLock.RLock()
		defer n.skillResponseLock.RUnlock()

		ctx.ResponseWriter.WriteHeader(http.StatusOK)
		ctx.ResponseWriter.Write(n.skillResponseData)
	}
}
