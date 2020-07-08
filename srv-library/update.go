package srvlibrary

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"sangjihaksik/share"

	"github.com/PuerkitoBio/goquery"
	skill "github.com/RyuaNerin/go-kakaoskill/v2"
	"github.com/getsentry/sentry-go"
)

var (
	regExctractSeatNumber = regexp.MustCompile(`reading_select_seat\('[^']+','(\d+)'`)
	regExtractSeatUsing   = regexp.MustCompile(`var\s+tbl_seat_id\s+=\s+'\d+\D(\d+)'`)
)

type roomData struct {
	once sync.Once

	Name     string
	PostData []byte
	Template string
	WebUrl   string

	enabled bool // false 일 때 : 운영시간 아님 혹은

	textBuffer bytes.Buffer

	seat map[int]SeatState // 좌석 정보

	// 여기서부터는 웹에서 쓸 부분
	webLock       sync.RWMutex
	webBody       []byte       // 웹뷰 데이터
	webBodyBuffer bytes.Buffer // 웹뷰 버퍼
	webETag       string

	webUpdateBuffer bytes.Buffer // 업데이트할 떄 HTML 메모리에 읽을 때 사용할 버퍼
}

type SeatState struct {
	SeatNum string
	Using   bool
}

var (
	skillData share.SkillData

	roomMap = map[int]*roomData{
		제1열람실: {
			Name:     "제 1 열람실 (3층)",
			PostData: share.ToBytes("sloc_code=SJU&group_code=0&reading_code=04"),
			Template: "room1.tmpl.htm",
			WebUrl:   fmt.Sprintf("%s%s?key=%d", share.ServerUri, pathWebView, 제1열람실),
		},
		제2열람실: {
			Name:     "제 2 열람실 (5층)",
			PostData: share.ToBytes("sloc_code=SJU&group_code=0&reading_code=05"),
			Template: "room2.tmpl.htm",
			WebUrl:   fmt.Sprintf("%s%s?key=%d", share.ServerUri, pathWebView, 제2열람실),
		},
		제3열람실A: {
			Name:     "제 3 열람실 A (5층)",
			PostData: share.ToBytes("sloc_code=SJU&group_code=0&reading_code=01"),
			Template: "room3a.tmpl.htm",
			WebUrl:   fmt.Sprintf("%s%s?key=%d", share.ServerUri, pathWebView, 제3열람실A),
		},
		제3열람실B: {
			Name:     "제 3 열람실 B (5층)",
			PostData: share.ToBytes("sloc_code=SJU&group_code=0&reading_code=07"),
			Template: "room3b.tmpl.htm",
			WebUrl:   fmt.Sprintf("%s%s?key=%d", share.ServerUri, pathWebView, 제3열람실B),
		},
		그룹스터디실: {
			Name:     "그룹스터디실(2층)",
			PostData: share.ToBytes("sloc_code=SJU&group_code=0&reading_code=06"),
			Template: "roomgroup.tmpl.htm",
			WebUrl:   fmt.Sprintf("%s%s?key=%d", share.ServerUri, pathWebView, 그룹스터디실),
		},
	}

	tg = template.Must(template.ParseGlob("srv-library/template/*.tmpl.htm"))
)

func init() {
	share.DoUpdate(share.Config.UpdatePeriodLibrary, update)
}

func update() {
	now := time.Now()

	var w sync.WaitGroup
	if updateTotal(now) {
		for _, s := range roomMap {
			w.Add(1)
			go s.update(&w, now)
		}
		w.Wait()
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

	res, err := share.Client.Do(req)
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

	sr := skill.SkillResponse{
		Version: "2.0",
		Template: skill.SkillTemplate{
			Outputs: []skill.Component{
				{
					ListCard: &skill.ListCard{
						Header: skill.ListItemHeader{
							Title: "열람실 좌석 정보",
						},
					},
				},
			},
		},
	}

	doc.Find("div.facility_box_whole > div").Each(
		func(index int, s *goquery.Selection) {
			var d *roomData
			var k int

			ff := strings.TrimSpace(s.Find("div.facility_box_head").Text())
			switch ff {
			case "제1열람실(3층)":
				k = 제1열람실
			case "제2열람실(5층)":
				k = 제2열람실
			case "제3열람실A(5층)":
				k = 제3열람실A
			case "제3열람실B(5층)":
				k = 제3열람실B
			case "그룹스터디실(2층)":
				k = 그룹스터디실
			default:
				return
			}
			d = roomMap[k]

			/**
			이용 가능 : 200 / 210

			2020년 2월 25일 토요일
			오전 1시 0분 기준
			//////////////////////////////
			오늘은 운영하지 않습니다

			2020년 2월 25일 토요일
			오전 1시 0분 기준
			*/

			sb := &d.textBuffer
			sb.Reset()

			// check disabled
			if disabled := s.Find("div.facility_disable_message"); len(disabled.Nodes) > 0 {
				d.enabled = false

				msg := strings.TrimSpace(disabled.Text())
				fmt.Fprint(sb, msg)

				d.makeTemplate(now, msg)
			} else {
				d.enabled = true
				seatPossible := s.Find("span.facility_box_seat_possiblenum").Text()
				seatWhole := s.Find("span.facility_box_seat_wholenum").Text()
				fmt.Fprintf(sb, "이용 가능 : %s / %s", seatPossible, seatWhole)
			}
		},
	)

	for _, i := range roomIndex {
		d := roomMap[i]
		if d.enabled {
			sr.Template.Outputs[0].ListCard.Items = append(
				sr.Template.Outputs[0].ListCard.Items,
				skill.ListItemItems{
					Title:       d.Name,
					Description: share.ToString(d.textBuffer.Bytes()),
					Link: skill.Link{
						Web: d.WebUrl,
					},
				},
			)
		} else {
			sr.Template.Outputs[0].ListCard.Items = append(
				sr.Template.Outputs[0].ListCard.Items,
				skill.ListItemItems{
					Title:       d.Name,
					Description: share.ToString(d.textBuffer.Bytes()),
				},
			)
		}
	}

	skillData.Update(&sr)

	return true
}

func updateTotalIsLogined() bool {
	req, _ := http.NewRequest("GET", "http://library.sangji.ac.kr/reading_lib_list.mir", nil)
	req.Header = http.Header{
		"User-Agent": []string{share.UserAgent},
	}

	res, err := share.Client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return false
	}
	res.Body.Close()

	return strings.Contains(res.Header.Get("Location"), "reading_reading_list")
}

func updateTotalLogin() bool {
	if !share.Login() {
		return false
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////
	// Visit page
	req, _ := http.NewRequest("GET", "https://library.sangji.ac.kr/", nil)
	req.Header = http.Header{
		"User-Agent": []string{share.UserAgent},
	}

	res, err := share.Client.Do(req)
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

	res, err = share.Client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return false
	}
	res.Body.Close()

	return true
}

func (m *roomData) update(w *sync.WaitGroup, now time.Time) {
	defer w.Done()

	m.once.Do(func() {
		m.seat = make(map[int]SeatState, 300)
	})

	req, _ := http.NewRequest("POST", "https://library.sangji.ac.kr/reading_seat_map.mir", bytes.NewReader(m.PostData))
	req.Header = http.Header{
		"User-Agent":   []string{share.UserAgent},
		"Content-Type": []string{"application/x-www-form-urlencoded"},
		"Referer":      []string{"https://library.sangji.ac.kr/reading_seat_map.mir"},
	}

	res, err := share.Client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return
	}
	defer res.Body.Close()

	m.webUpdateBuffer.Reset()
	_, err = io.Copy(&m.webUpdateBuffer, res.Body)
	if err != nil && err != io.EOF {
		sentry.CaptureException(err)
		return
	}

	body := share.ToString(m.webUpdateBuffer.Bytes())

	// 비우기
	for k := range m.seat {
		delete(m.seat, k)
	}

	// 좌석 정보 읽는 부분
	for _, match := range regExctractSeatNumber.FindAllStringSubmatch(body, -1) {
		seatNumStr := match[1]
		seatNum, err := strconv.Atoi(seatNumStr)
		if err != nil {
			continue
		}

		m.seat[seatNum] = SeatState{
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

		m.seat[seatNum] = SeatState{
			SeatNum: seatNumStr,
			Using:   true,
		}
	}

	m.makeTemplate(now, "")
}

func (d *roomData) makeTemplate(now time.Time, disabledMessage string) {
	type templateData struct {
		Name      string // 열람실 이름
		UpdatedAt string // 업데이트 기준일

		DisabledMessage string // 에러용 메시지

		Seat map[int]SeatState
	}

	td := templateData{
		Name:      d.Name,
		UpdatedAt: share.TimeFormatKr.Replace(now.Format("2006년 1월 2일 Mon pm 3시 4분 기준")),
	}

	var templateFile string
	if disabledMessage == "" {
		td.Seat = d.seat
		templateFile = d.Template
	} else {
		td.DisabledMessage = disabledMessage
		templateFile = "disabled.tmpl.htm"
	}

	h := fnv.New64()

	d.webLock.Lock()
	defer d.webLock.Unlock()

	d.webBody = nil

	d.webBodyBuffer.Reset()
	err := tg.ExecuteTemplate(io.MultiWriter(&d.webBodyBuffer, h), templateFile, td)
	if err != nil {
		sentry.CaptureException(err)
		return
	}

	d.webBody = d.webBodyBuffer.Bytes()
	d.webETag = hex.EncodeToString(h.Sum(nil))
}
