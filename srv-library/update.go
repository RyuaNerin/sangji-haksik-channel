package srvlibrary

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"sangjihaksik/share"

	"github.com/PuerkitoBio/goquery"
	"github.com/getsentry/sentry-go"
)

var (
	clientJar, _ = cookiejar.New(nil)
	client       = http.Client{
		Transport: share.FiddlerTransport(new(http.Transport)),
		Jar:       clientJar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	loginPostData = url.Values{
		"siteCode":  []string{"kor"},
		"returnUrl": []string{""},
		"id":        []string{share.Config.Id},
		"password":  []string{share.Config.Pw},
	}.Encode()

	regExctractSeatNumber = regexp.MustCompile(`reading_select_seat\('[^']+','(\d+)'`)
	regExtractSeatUsing   = regexp.MustCompile(`var\s+tbl_seat_id\s+=\s+'\d+\D(\d+)'`)
)

type data struct {
	key              string
	name             string
	updatePostData   []byte
	templateFileName string

	lock             sync.RWMutex
	enabled          bool         // false 일 때 : 운영시간 아님 혹은
	skillHtmlLink    string       // 스킬 전송할 때 보낼 주소
	skillText        string       // 사전 생성된 카카오톡 전용 문자열
	skillTextBuilder bytes.Buffer // skillText 전용 버퍼

	webViewbody   []byte       // 웹뷰 데이터
	webViewBuffer bytes.Buffer // 웹뷰 버퍼

	updateHtmlBuffer bytes.Buffer             // 업데이트할 떄 HTML 메모리에 읽을 때 사용할 버퍼
	updateMapBuffer  map[int]templateDataSeat // 돌려쓰기

	updating int32
}

var (
	seat1    = newDataSeat("sloc_code=SJU&group_code=0&reading_code=04", "0", "room1.tmpl.htm", "제 1 열람실 (3층)")
	seat2    = newDataSeat("sloc_code=SJU&group_code=0&reading_code=05", "1", "room2.tmpl.htm", "제 2 열람실 (5층)")
	seat3a   = newDataSeat("sloc_code=SJU&group_code=0&reading_code=03", "2", "room3a.tmpl.htm", "제 3 열람실 A (5층)") // TODO
	seat3b   = newDataSeat("sloc_code=SJU&group_code=0&reading_code=07", "3", "room3b.tmpl.htm", "제 3 열람실 B (5층)") // TODO
	seatRoom = newDataSeat("sloc_code=SJU&group_code=0&reading_code=06", "4", "roomgroup.tmpl.htm", "그룹스터디실(2층)")
)

func newDataSeat(postData string, key string, templateName string, name string) data {
	return data{
		key:              key,
		name:             name,
		updatePostData:   []byte(postData),
		templateFileName: templateName,

		updateMapBuffer: make(map[int]templateDataSeat, 300),
	}
}

func updateFunc() {
	ticker := time.NewTicker(share.Config.UpdatePeriodLibrary)

	for {
		now := time.Now()

		if updateTotal(now) {
			go seat1.update(now)
			go seat2.update(now)
			go seat3a.update(now)
			go seat3b.update(now)
			go seatRoom.update(now)
		}

		<-ticker.C
	}
}

func updateTotal(now time.Time) bool {
	if !updateTotalIsLogined() {
		if !updateTotalLogin() {
			return false
		}

		if !updateTotalIsLogined() {
			return false
		}
	}

	req, _ := http.NewRequest("GET", "https://library.sangji.ac.kr/reading_reading_list.mir", nil)
	req.Header = http.Header{
		"User-Agent": []string{share.UserAgent},
	}

	res, err := client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return false
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil && err != io.EOF {
		sentry.CaptureException(err)
		return false
	}

	seat1.lock.Lock()
	seat2.lock.Lock()
	seat3a.lock.Lock()
	seat3b.lock.Lock()

	defer seat1.lock.Unlock()
	defer seat2.lock.Unlock()
	defer seat3a.lock.Unlock()
	defer seat3b.lock.Unlock()

	doc.Find("div.facility_box_whole > div").Each(
		func(index int, s *goquery.Selection) {
			var d *data

			ff := strings.TrimSpace(s.Find("div.facility_box_head").Text())
			switch ff {
			case "제1열람실(3층)":
				d = &seat1
			case "제2열람실(5층)":
				d = &seat2
			case "제3열람실A(5층)":
				d = &seat3a
			case "제3열람실B(5층)":
				d = &seat3b
			case "그룹스터디실(2층)":
				d = &seatRoom
			default:
				return
			}

			d.skillHtmlLink = fmt.Sprintf("%s%s?key=%s&for-cache=%d", share.ServerUri, pathWebView, d.key, now.Unix())

			/**
			이용 가능 : 200 / 210

			2020년 2월 25일 토요일
			오전 1시 0분 기준
			//////////////////////////////
			오늘은 운영하지 않습니다

			2020년 2월 25일 토요일
			오전 1시 0분 기준
			*/

			sb := &d.skillTextBuilder
			sb.Reset()

			// check disabled
			if disabled := s.Find("div.facility_disable_message"); len(disabled.Nodes) > 0 {
				d.enabled = false

				msg := strings.TrimSpace(disabled.Text())
				fmt.Fprintln(sb, msg)

				d.makeTemplateError(now, msg)
			} else {
				d.enabled = true
				seatPossible := s.Find("span.facility_box_seat_possiblenum").Text()
				seatWhole := s.Find("span.facility_box_seat_wholenum").Text()
				fmt.Fprintf(sb, "이용 가능 : %s / %s", seatPossible, seatWhole)
				fmt.Fprintln(sb)
			}

			fmt.Fprintln(sb)
			fmt.Fprintln(sb, share.TimeFormatKr.Replace(now.Format("2006년 1월 2일 Mon")))
			fmt.Fprint(sb, share.TimeFormatKr.Replace(now.Format("pm 3시 4분 기준")))

			d.skillText = share.ToString(sb.Bytes())
		},
	)

	return true
}

func updateTotalIsLogined() bool {
	req, _ := http.NewRequest("GET", "http://library.sangji.ac.kr/reading_lib_list.mir", nil)
	req.Header = http.Header{
		"User-Agent": []string{share.UserAgent},
	}

	res, err := client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return false
	}
	res.Body.Close()

	return strings.Contains(res.Header.Get("Location"), "reading_reading_list")
}

func updateTotalLogin() bool {
	// VisitPage
	req, _ := http.NewRequest("GET", "https://www.sangji.ac.kr/prog/login/actionSangjiLogin.do", nil)
	req.Header = http.Header{
		"User-Agent": []string{share.UserAgent},
	}
	res, err := client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return false
	}
	res.Body.Close()

	// Login
	// https://www.sangji.ac.kr/prog/login/actionSangjiLogin.do
	// siteCode=kor&returnUrl=&id=********&password=*********
	req, _ = http.NewRequest("POST", "https://www.sangji.ac.kr/prog/login/actionSangjiLogin.do", strings.NewReader(loginPostData))
	req.Header = http.Header{
		"User-Agent":   []string{share.UserAgent},
		"Content-Type": []string{"application/x-www-form-urlencoded"},
	}

	res, err = client.Do(req)
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

	////////////////////////////////////////////////////////////////////////////////////////////////////
	// Visit page
	req, _ = http.NewRequest("GET", "https://library.sangji.ac.kr/", nil)
	req.Header = http.Header{
		"User-Agent": []string{share.UserAgent},
	}

	res, err = client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return false
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(share.FixEncoding(res))
	if err != nil && err != io.EOF {
		sentry.CaptureException(err)
		return false
	}

	actionRaw, ok := doc.Find("form").Attr("action")
	if !ok {
		return false
	}

	actionUrl, _ := url.Parse("https://library.sangji.ac.kr/")
	actionUrl, err = actionUrl.Parse(actionRaw)
	if err != nil {
		return false
	}

	postData := url.Values{}
	doc.Find("form input").Each(
		func(index int, s *goquery.Selection) {
			name, ok := s.Attr("name")
			if !ok {
				return
			}

			value, ok := s.Attr("value")
			if !ok {
				return
			}

			postData.Add(name, value)
		},
	)

	// Post
	req, _ = http.NewRequest("POST", actionUrl.String(), strings.NewReader(postData.Encode()))
	req.Header = http.Header{
		"User-Agent":   []string{share.UserAgent},
		"Content-Type": []string{"application/x-www-form-urlencoded;charset=utf-8"},
		"Referer":      []string{"https://library.sangji.ac.kr/"},
	}

	res, err = client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return false
	}
	res.Body.Close()

	return true
}

func (m *data) update(bgnde time.Time) {
	m.lock.RLock()
	if !m.enabled {
		m.lock.RUnlock()
		return
	}
	m.lock.RUnlock()

	if !atomic.CompareAndSwapInt32(&m.updating, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&m.updating, 0)

	now := time.Now()

	req, _ := http.NewRequest("POST", "https://library.sangji.ac.kr/reading_seat_map.mir", bytes.NewReader(m.updatePostData))
	req.Header = http.Header{
		"User-Agent":   []string{share.UserAgent},
		"Content-Type": []string{"application/x-www-form-urlencoded"},
		"Referer":      []string{"https://library.sangji.ac.kr/reading_seat_map.mir"},
	}

	res, err := client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return
	}
	defer res.Body.Close()

	m.updateHtmlBuffer.Reset()
	_, err = io.Copy(&m.updateHtmlBuffer, res.Body)
	if err != nil && err != io.EOF {
		sentry.CaptureException(err)
		return
	}

	body := share.ToString(m.updateHtmlBuffer.Bytes())

	// 비우기
	for k := range m.updateMapBuffer {
		delete(m.updateMapBuffer, k)
	}

	// 좌석 정보 읽는 부분
	for _, match := range regExctractSeatNumber.FindAllStringSubmatch(body, -1) {
		seatNumStr := match[1]
		seatNum, err := strconv.Atoi(seatNumStr)
		if err != nil {
			continue
		}

		m.updateMapBuffer[seatNum] = templateDataSeat{
			SeatNum: seatNumStr,
		}
	}

	// 사용중인 좌석 체크하는 부분
	for _, match := range regExtractSeatUsing.FindAllStringSubmatch(body, -1) {
		seatNumStr := match[1]
		seatNum, err := strconv.Atoi(seatNumStr)
		if err != nil {
			continue
		}

		m.updateMapBuffer[seatNum] = templateDataSeat{
			SeatNum: seatNumStr,
			Using:   true,
		}
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	m.makeTemplate(now)
}
