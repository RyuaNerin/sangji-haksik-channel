package srvbus

import (
	"net/http"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

func skillHandler(ctx *skill.Context) {
	var d *routeInfo

	if ctx.Payload.Action.Params != nil {
		if keyRaw, ok := ctx.Payload.Action.Params["key"]; ok {
			if key, ok := keyRaw.(string); ok {
				d, _ = routeList[key]
			}
		}
	}

	if d == nil {
		ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
		return
	}

	d.skillResponseLock.RLock()
	defer d.skillResponseLock.RUnlock()

	ctx.ResponseWriter.WriteHeader(http.StatusOK)
	ctx.ResponseWriter.Write(d.skillResponseBody)
}
