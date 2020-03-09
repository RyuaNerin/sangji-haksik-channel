package srvlibrary

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"sangjihaksik/share"

	"github.com/PuerkitoBio/goquery"
	skill "github.com/RyuaNerin/go-kakaoskill/v2"
	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
)

var (
	regExctractSeatNumber = regexp.MustCompile(`reading_select_seat\('[^']+','(\d+)'`)
	regExtractSeatUsing   = regexp.MustCompile(`var\s+tbl_seat_id\s+=\s+'\d+\D(\d+)'`)
)

type data struct {
	name             string
	updatePostData   []byte
	templateFileName string

	lock    sync.RWMutex
	enabled bool // false 일 때 : 운영시간 아님 혹은

	textBuffer          bytes.Buffer
	skillResponse       []byte       // 스킬 응답 사전 생성
	skillResponseBuffer bytes.Buffer // skillResponse 버퍼

	webBody            []byte       // 웹뷰 데이터
	webBodyBuffer      bytes.Buffer // 웹뷰 버퍼
	webLastModified    string       // share.ToString(webViewETagBuf)
	webLastModifiedBuf []byte

	updateHtmlBuffer bytes.Buffer             // 업데이트할 떄 HTML 메모리에 읽을 때 사용할 버퍼
	updateMapBuffer  map[int]templateDataSeat // 돌려쓰기

	updating int32

	// 메모리 재할당 방지용 변수
	skillResponseWithoutButton skill.SkillResponse
	skillResponseWithButton    skill.SkillResponse
}

var (
	seat1    = newDataSeat("sloc_code=SJU&group_code=0&reading_code=04", "0", "room1.tmpl.htm", "제 1 열람실 (3층)")
	seat2    = newDataSeat("sloc_code=SJU&group_code=0&reading_code=05", "1", "room2.tmpl.htm", "제 2 열람실 (5층)")
	seat3a   = newDataSeat("sloc_code=SJU&group_code=0&reading_code=01", "2", "room3a.tmpl.htm", "제 3 열람실 A (5층)")
	seat3b   = newDataSeat("sloc_code=SJU&group_code=0&reading_code=07", "3", "room3b.tmpl.htm", "제 3 열람실 B (5층)")
	seatRoom = newDataSeat("sloc_code=SJU&group_code=0&reading_code=06", "4", "roomgroup.tmpl.htm", "그룹스터디실(2층)")
)

func newDataSeat(postData string, key string, templateName string, name string) data {
	return data{
		name:             name,
		updatePostData:   []byte(postData),
		templateFileName: templateName,

		updateMapBuffer: make(map[int]templateDataSeat, 300),

		skillResponseWithButton: skill.SkillResponse{
			Version: "2.0",
			Template: skill.SkillTemplate{
				Outputs: []skill.Component{
					skill.Component{
						BasicCard: &skill.BasicCard{
							Title: name,
							Buttons: []skill.Button{
								skill.Button{
									Label:      "좌석 보기",
									Action:     "webLink",
									WebLinkUrl: fmt.Sprintf("%s%s?key=%s", share.ServerUri, pathWebView, key),
								},
							},
						},
					},
				},
				QuickReplies: baseReplies,
			},
		},
		skillResponseWithoutButton: skill.SkillResponse{
			Version: "2.0",
			Template: skill.SkillTemplate{
				Outputs: []skill.Component{
					skill.Component{
						BasicCard: &skill.BasicCard{
							Title: name,
						},
					},
				},
				QuickReplies: baseReplies,
			},
		},
	}
}

func updateFunc() {
	// 첫갱신 이후 단위 맞추기
	var ticker *time.Ticker

	firstRun := true
	for {
		now := time.Now()

		if updateTotal(now) {
			go seat1.update()
			go seat2.update()
			go seat3a.update()
			go seat3b.update()
			go seatRoom.update()
		}

		if firstRun {
			firstRun = false

			<-time.After(time.Until(time.Now().Truncate(share.Config.UpdatePeriodLibrary).Add(share.Config.UpdatePeriodLibrary)))
			ticker = time.NewTicker(share.Config.UpdatePeriodLibrary)
		} else {
			<-ticker.C
		}
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

			// Skill Response 생성

			var res *skill.SkillResponse
			if d.enabled {
				res = &d.skillResponseWithButton
			} else {
				res = &d.skillResponseWithoutButton
			}
			res.Template.Outputs[0].BasicCard.Description = share.ToString(sb.Bytes())

			d.skillResponseBuffer.Reset()
			err := jsoniter.NewEncoder(&d.skillResponseBuffer).Encode(res)
			if err != nil {
				d.skillResponse = responseError
				sentry.CaptureException(err)
				return
			}

			d.skillResponse = d.skillResponseBuffer.Bytes()
		},
	)

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

func (m *data) update() {
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

	res, err := share.Client.Do(req)
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

	m.updateETag(now)
}
