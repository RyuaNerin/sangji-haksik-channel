package main

import (
	"net"
	"net/http"
	"os"
	"runtime"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

const (
	AddrWin   = ":5577"
	AddrLinux = "/run/sangji-haksik-channel/sock"
)

func startWebhook() {
	mux := http.ServeMux{}
	mux.HandleFunc(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		},
	)

	if runtime.GOOS == "windows" {
		mux.HandleFunc("/today", skill.F(handleToday))
	} else {
		mux.HandleFunc("/today", skill.FP("X-Real-IP", handleToday))
	}

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
		Handler: &mux,
	}

	go server.Serve(l)
}
