package share

import (
	"bytes"
	"io"
	"net/http"

	"golang.org/x/net/html/charset"
)

// 제발 utf-8 해주세요
func FixEncoding(resp *http.Response) io.Reader {
	rdBodyF := bytes.NewBuffer(nil)
	rdBody := io.MultiReader(rdBodyF, resp.Body)

	rdTee := io.TeeReader(resp.Body, rdBodyF)

	data := make([]byte, 1024)
	if _, err := rdTee.Read(data); err == nil || err == io.EOF {
		if e, _, _ := charset.DetermineEncoding(data, resp.Header.Get("Content-Type")); e != nil {
			rdBody = e.NewDecoder().Reader(rdBody)
		}
	}

	return rdBody
}
