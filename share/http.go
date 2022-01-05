package share

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

func NewHttpClient() *http.Client {
	t := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 3 * time.Second,
	}
	c := &http.Client{
		Timeout:   30 * time.Second,
		Transport: t,
	}

	if tcpProxy, err := net.DialTimeout("tcp", Config.Fiddler, time.Second); err == nil {
		tcpProxy.Close()

		u, _ := url.Parse("http://" + Config.Fiddler)
		t.Proxy = http.ProxyURL(u)
	}

	return c
}

func Login(client *http.Client, id string, pw string) bool {
	// VisitPage
	// https://www.sangji.ac.kr/kor/login.do
	req, _ := http.NewRequest("GET", "https://www.sangji.ac.kr/kor/login.do", nil)
	req.Header = http.Header{
		"User-Agent": []string{UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return false
	}
	io.Copy(io.Discard, res.Body)
	res.Body.Close()

	// Login
	// https://www.sangji.ac.kr/prog/login/actionSangjiLogin.do
	// siteCode=kor&returnUrl=&id=********&password=*********
	loginPostData := ToBytes(url.Values{
		"siteCode":  []string{"kor"},
		"returnUrl": []string{""},
		"id":        []string{Config.Id},
		"password":  []string{Config.Pw},
	}.Encode())

	req, _ = http.NewRequest("POST", "https://www.sangji.ac.kr/prog/login/actionSangjiLogin.do", bytes.NewReader(loginPostData))
	req.Header = http.Header{
		"User-Agent":   []string{UserAgent},
		"Content-Type": []string{"application/x-www-form-urlencoded"},
	}

	res, err = client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return false
	}
	io.Copy(io.Discard, res.Body)
	res.Body.Close()

	loc, err := res.Location()
	if err != nil {
		sentry.CaptureException(err)
		return false
	}

	if strings.Contains(loc.Path, "login.do") {
		sentry.CaptureException(errors.New(loc.Query().Get("message")))
		return false
	}

	return true
}
