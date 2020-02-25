package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
)

type KakaoContext struct {
	gin *gin.Context
	rd  *chatBotRequest
}

type KakaoHandler func(ctx KakaoContext)

const (
	AddrWin   = ":5577"
	AddrLinux = "/run/sangji-haksik-channel/sock"
)

func init() {
	if runtime.GOOS != "windows" {
		gin.SetMode(gin.ReleaseMode)
	}
}

func startWebhook() {
	e := gin.New()
	e.Use(ginRecovery)
	e.POST("/today", handlePOST(handleToday))

	var l net.Listener
	var err error

	if runtime.GOOS == "windows" {
		l, err = net.Listen("tcp", AddrWin)
	} else {
		if _, err := os.Stat(AddrLinux); !os.IsNotExist(err) {
			err := os.Remove(AddrLinux)
			if err != nil {
				panic(err)
			}
		}

		l, err = net.Listen("unix", AddrLinux)
		if err != nil {
			panic(err)
		}
		err = os.Chmod(AddrLinux, 0777)
	}
	if err != nil {
		panic(err)
	}

	server := http.Server{
		Handler: e,
	}

	go server.Serve(l)
}

func ginRecovery(ctx *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			var brokenPipe bool
			if ne, ok := err.(*net.OpError); ok {
				if se, ok := ne.Err.(*os.SyscallError); ok {
					if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
						brokenPipe = true
					}
				}
			}

			if _, ok := err.(*net.OpError); !ok {
				sentry.CaptureException(err.(error))
			}

			if brokenPipe {
				_ = ctx.Error(err.(error))
				ctx.Abort()
			} else {
				ctx.AbortWithStatus(http.StatusInternalServerError)
			}
		}
	}()
	ctx.Next()
}

func handlePOST(fn KakaoHandler) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		addr := ctx.GetHeader("X-Real-IP")
		if addr == "" {
			addr = ctx.Request.RemoteAddr
		}

		if host, _, err := net.SplitHostPort(addr); err == nil {
			addr = host
		}

		switch addr {
		case "219.249.231.40":
		case "219.249.231.41":
		case "219.249.231.42":
		default:
			ctx.Status(http.StatusBadRequest)
			return
		}

		var rd chatBotRequest
		err := json.NewDecoder(ctx.Request.Body).Decode(&rd)
		if err != nil {
			sentry.CaptureException(err)
			ctx.Status(http.StatusBadRequest)
			return
		}

		fn(
			KakaoContext{
				gin: ctx,
				rd:  &rd,
			},
		)
	}
}

func (ctx *KakaoContext) WriteSimpleText(str string) {
	res := chatBotResponse{
		Version: "2.0",
		Template: skillTemplate{
			Outputs: []component{
				component{
					SimpleText: &simpleText{
						Text: str,
					},
				},
			},
		},
	}

	ctx.gin.Status(http.StatusOK)
	_ = jsoniter.NewEncoder(ctx.gin.Writer).Encode(&res)
}
