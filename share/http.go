package share

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

var (
	// snagji.ac.kr 접근용 클라이언트
	Client *http.Client

	loginPostData []byte
)

func init() {
	http.DefaultTransport = fiddlerTransport(http.DefaultTransport)

	jar, _ := cookiejar.New(nil)
	Client = &http.Client{
		Transport: fiddlerTransport(new(http.Transport)),
		Jar:       jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	loginPostData = ToBytes(url.Values{
		"siteCode":  []string{"kor"},
		"returnUrl": []string{""},
		"id":        []string{Config.Id},
		"password":  []string{Config.Pw},
	}.Encode())

}

func fiddlerTransport(rt http.RoundTripper) http.RoundTripper {
	if rt == nil {
		return nil
	}

	if htp, ok := rt.(*http.Transport); ok {
		if tcpProxy, err := net.DialTimeout("tcp", Config.Fiddler, time.Second); err == nil {
			tcpProxy.Close()

			u, _ := url.Parse("http://" + Config.Fiddler)
			htp.Proxy = http.ProxyURL(u)
		}
	}

	return rt
}

var loginBufferPool bytes.Buffer

func Login() bool {
	// VisitPage
	// https://www.sangji.ac.kr/prog/login/actionSangjiLogin.do
	req, _ := http.NewRequest("GET", "https://www.sangji.ac.kr/prog/login/actionSangjiLogin.do", nil)
	req.Header = http.Header{
		"User-Agent": []string{UserAgent},
	}
	res, err := Client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return false
	}
	res.Body.Close()

	// Login
	// https://www.sangji.ac.kr/prog/login/actionSangjiLogin.do
	// siteCode=kor&returnUrl=&id=********&password=*********
	req, _ = http.NewRequest("POST", "https://www.sangji.ac.kr/prog/login/actionSangjiLogin.do", bytes.NewReader(loginPostData))
	req.Header = http.Header{
		"User-Agent":   []string{UserAgent},
		"Content-Type": []string{"application/x-www-form-urlencoded"},
	}

	res, err = Client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return false
	}
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
