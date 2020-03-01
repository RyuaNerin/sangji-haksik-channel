package share

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

func init() {
	http.DefaultTransport = FiddlerTransport(http.DefaultTransport)
}

func FiddlerTransport(rt http.RoundTripper) http.RoundTripper {
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
