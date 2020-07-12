package srvlibrary

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"html/template"
	"io"
	"net/http"
	"net/http/cookiejar"
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
	regExctractSeatNumber  = regexp.MustCompile(`reading_select_seat\(\s*["'][^']+["']\s*,\s*["'](\d+)["']\s*(:?,\s*([^,]+?)\s*,\s*([^,]+?)\s*)?\)`)
	regExtractSeatUsing    = regexp.MustCompile(`var\s+tbl_seat_id\s*=\s*["']\d+\D(\d+)["']`)
	regExtractSeatDisabled = regexp.MustCompile(`remove_onclick_attr\(\s*['"]\d+\D(\d+)['"]\s*\)`)
	regY                   = regexp.MustCompile(`['"]Y['"]`)
)

type roomData struct {
	Name     string
	PostData []byte
	Template string
	WebUrl   string
	PngUrl   string

	// Png
	Image *imageData

	once sync.Once

	enabled bool // false 일 때 : 운영시간 아님 혹은

	textBuffer bytes.Buffer

	seat map[int]SeatState // 좌석 정보

	// 여기서부터는 웹에서 쓸 부분
	webLock       sync.RWMutex
	webBody       []byte       // 웹뷰 데이터
	webBodyBuffer bytes.Buffer // 웹뷰 버퍼
	webETag       string

	responseHash      uint64
	reponseBodyBuffer bytes.Buffer // 업데이트할 떄 HTML 메모리에 읽을 때 사용할 버퍼
}

type SeatState struct {
	SeatNum    string
	Disabled   bool
	NoteBook   bool
	WheelChair bool

	Using bool
}

var (
	skillData share.SkillData

	roomMap = map[int]*roomData{
		제1열람실: {
			Name:     "제 1 열람실 (3층)",
			PostData: share.ToBytes("sloc_code=SJU&group_code=0&reading_code=04"),
			Template: "room1.tmpl.htm",
			WebUrl:   fmt.Sprintf("%s%s?key=%d", share.ServerUri, pathWebView, 제1열람실),
			PngUrl:   fmt.Sprintf("%s%s?key=%d&type=png", share.ServerUri, pathWebView, 제1열람실),
			Image:    newImageData("room1.json"),
		},
		제2열람실: {
			Name:     "제 2 열람실 (5층)",
			PostData: share.ToBytes("sloc_code=SJU&group_code=0&reading_code=05"),
			Template: "room2.tmpl.htm",
			WebUrl:   fmt.Sprintf("%s%s?key=%d", share.ServerUri, pathWebView, 제2열람실),
			PngUrl:   fmt.Sprintf("%s%s?key=%d&type=png", share.ServerUri, pathWebView, 제2열람실),
			Image:    newImageData("room2.json"),
		},
		제3열람실A: {
			Name:     "제 3 열람실 A (5층)",
			PostData: share.ToBytes("sloc_code=SJU&group_code=0&reading_code=01"),
			Template: "room3a.tmpl.htm",
			WebUrl:   fmt.Sprintf("%s%s?key=%d", share.ServerUri, pathWebView, 제3열람실A),
			PngUrl:   fmt.Sprintf("%s%s?key=%d&type=png", share.ServerUri, pathWebView, 제3열람실A),
			Image:    newImageData("room3a.json"),
		},
		제3열람실B: {
			Name:     "제 3 열람실 B (5층)",
			PostData: share.ToBytes("sloc_code=SJU&group_code=0&reading_code=07"),
			Template: "room3b.tmpl.htm",
			WebUrl:   fmt.Sprintf("%s%s?key=%d", share.ServerUri, pathWebView, 제3열람실B),
			PngUrl:   fmt.Sprintf("%s%s?key=%d&type=png", share.ServerUri, pathWebView, 제3열람실B),
			Image:    newImageData("room3b.json"),
		},
		그룹스터디실: {
			Name:     "그룹스터디실(2층)",
			PostData: share.ToBytes("sloc_code=SJU&group_code=0&reading_code=06"),
			Template: "roomgroup.tmpl.htm",
			WebUrl:   fmt.Sprintf("%s%s?key=%d", share.ServerUri, pathWebView, 그룹스터디실),
			PngUrl:   fmt.Sprintf("%s%s?key=%d&type=png", share.ServerUri, pathWebView, 그룹스터디실),
			Image:    newImageData("roomgroup.json"),
		},
	}

	tg = template.Must(template.ParseGlob("srv-library/public/*.tmpl.htm"))

	client = http.Client{
		Transport: http.DefaultTransport,
		Jar: func() *cookiejar.Jar {
			j, _ := cookiejar.New(nil)
			return j
		}(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
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

	listCardItems := make([]skill.ListItemItems, 0, 5)
	for _, i := range roomIndex {
		d := roomMap[i]

		item := skill.ListItemItems{
			Title:       d.Name,
			Description: share.ToString(d.textBuffer.Bytes()),
		}
		if d.enabled {
			item.ImageUrl = d.PngUrl
			item.Link = skill.Link{
				Web: d.WebUrl,
			}
		}

		listCardItems = append(listCardItems, item)
	}

	sr := skill.SkillResponse{
		Version: "2.0",
		Template: skill.SkillTemplate{
			Outputs: []skill.Component{
				{
					ListCard: &skill.ListCard{
						Header: skill.ListItemHeader{
							Title: "열람실 좌석 정보",
							Link: skill.Link{
								Web: "https://library.sangji.ac.kr/reading_lib_list.mir",
							},
						},
						Items: listCardItems,
					},
				},
			},
		},
	}

	skillData.Update(&sr)

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
	if !share.Login(&client, share.Config.Id, share.Config.Pw) {
		return false
	}

	////////////////////////////////////////////////////////////////////////////////////////////////////
	// Visit page
	req, _ := http.NewRequest("GET", "https://library.sangji.ac.kr/", nil)
	req.Header = http.Header{
		"User-Agent": []string{share.UserAgent},
	}

	res, err := client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return false
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return false
	}

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

	res, err := client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return
	}

	h := fnv.New64()

	m.reponseBodyBuffer.Reset()
	_, err = io.Copy(io.MultiWriter(&m.reponseBodyBuffer, h), res.Body)
	if err != nil && err != io.EOF {
		sentry.CaptureException(err)
		return
	}

	if m.responseHash == h.Sum64() {
		m.makeTemplate(now, "")
		return
	}

	body := share.ToString(m.reponseBodyBuffer.Bytes())

	// 좌석 정보 읽는 부분
	for _, match := range regExctractSeatNumber.FindAllStringSubmatch(body, -1) {
		seatNumStr := match[1]
		seatNum, err := strconv.Atoi(seatNumStr)
		if err != nil {
			continue
		}

		ss := SeatState{
			SeatNum: seatNumStr,
		}

		if len(match) > 2 {
			ss.WheelChair = regY.MatchString(match[2])
			ss.NoteBook = regY.MatchString(match[3])
		}

		m.seat[seatNum] = ss
	}

	// 사용중인 좌석 체크하는 부분
	for _, match := range regExtractSeatUsing.FindAllStringSubmatch(body, -1) {
		seatNumStr := match[1]
		seatNum, err := strconv.Atoi(seatNumStr)
		if err != nil {
			continue
		}

		ss, ok := m.seat[seatNum]
		if ok {
			ss.Using = true
			m.seat[seatNum] = ss
		}
	}

	// 비활성화 된 좌석
	for _, match := range regExtractSeatDisabled.FindAllStringSubmatch(body, -1) {
		seatNumStr := match[1]
		seatNum, err := strconv.Atoi(seatNumStr)
		if err != nil {
			continue
		}

		ss, ok := m.seat[seatNum]
		if ok {
			ss.Disabled = true
			m.seat[seatNum] = ss
		}
	}

	m.makeTemplate(now, "")
	m.Image.DrawImage(now, m.seat)
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
