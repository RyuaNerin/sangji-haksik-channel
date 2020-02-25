package main

import (
	"net/http"

	"github.com/getsentry/sentry-go"
)

func init() {
	err := sentry.Init(
		sentry.ClientOptions{
			Dsn:           "https://06bb81a9c10c411a99e33d32a7e35d7f@sentry.ryuar.in/14",
			HTTPTransport: &http.Transport{},
		},
	)
	if err != nil {
		panic(err)
	}
}
