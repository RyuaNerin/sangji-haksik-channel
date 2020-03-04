package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"sangjihaksik/share"
	_ "sangjihaksik/srv-bus"
	_ "sangjihaksik/srv-library"
	_ "sangjihaksik/srv-menu"
)

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	var l net.Listener
	var err error

	if _, err := net.ResolveTCPAddr("tcp", share.ListenAddr); err == nil {
		l, err = net.Listen("tcp", share.ListenAddr)
	} else {
		if _, err := os.Stat(share.ListenAddr); !os.IsNotExist(err) {
			err := os.Remove(share.ListenAddr)
			if err != nil {
				panic(err)
			}
		}

		l, err = net.Listen("unix", share.ListenAddr)
		if err != nil {
			panic(err)
		}
		err = os.Chmod(share.ListenAddr, 0777)
	}
	if err != nil {
		panic(err)
	}

	server := http.Server{
		Handler: share.HttpMux,
	}

	log.Println("Serve")
	go server.Serve(l)

	<-sig

	log.Println("Exit")
}
