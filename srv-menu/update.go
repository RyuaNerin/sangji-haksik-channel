package srvmenu

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"html"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"sangjihaksik/share"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
)

type data struct {
	Name string
	Url  string

	menu [5]share.SkillData // 월 ~ 금
	buf  strings.Builder    // 일일 시간표 텍스트 생성에 사용될 버퍼
	hash uint64             // 메뉴 API 호출 결과가 업데이트 되었는지 확인하는 용도.
}

var (
	menu = map[int]*data{
		민주학생: {
			Name: "민주관 학생식당",
			Url:  "https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_01/DS/getCalendar.do",
		},
		행복기숙: {
			Name: "민주관 교직원식당",
			Url:  "https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_05/DP/getCalendar.do",
		},
		창조학생: {
			Name: "창조관 학생식당",
			Url:  "https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_03/CS/getCalendar.do",
		},
		창조교직: {
			Name: "창조관 교직원식당",
			Url:  "https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_04/CP/getCalendar.do",
		},
	}

	httpClient = share.NewHttpClient()
)

func init() {
	share.DoUpdate(share.Config.UpdatePeriodMenu, update)
}

func update() {
	bgnde := time.Now()

	switch bgnde.Weekday() {
	case time.Saturday: // 토요일 업데이트 안함
		return
	case time.Sunday: // 내일 (월요일) 꺼 미리 업데이트
		bgnde.Add(24 * time.Hour)
	default:
		bgnde = bgnde.Add(time.Duration(bgnde.Weekday()-time.Monday) * -24 * time.Hour)
	}

	postData := []byte(bgnde.Format("bgnde=2006-01-02"))

	var w sync.WaitGroup
	for _, d := range menu {
		w.Add(1)
		go d.update(&w, bgnde, postData)
	}
	w.Wait()
}

func (d *data) update(w *sync.WaitGroup, bgnde time.Time, postData []byte) {
	defer w.Done()

	req, _ := http.NewRequest("POST", d.Url, bytes.NewReader(postData))
	req.Header = http.Header{
		"User-Agent":   []string{"sangji-haksik-channel"},
		"Content-Type": []string{"application/x-www-form-urlencoded; charset=utf-8"},
	}
	req.ContentLength = int64(len(postData))

	res, err := httpClient.Do(req)
	if err != nil {
		d.updateErr(bgnde)
		sentry.CaptureException(err)
		return
	}
	defer res.Body.Close()

	var responseJson struct {
		Item []struct {
			Type     string `json:"type"`
			Time     string `json:"time"`
			ModDate  string `json:"modDate"`
			WeekDay0 string `json:"menu1"`
			WeekDay1 string `json:"menu2"`
			WeekDay2 string `json:"menu3"`
			WeekDay3 string `json:"menu4"`
			WeekDay4 string `json:"menu5"`
		} `json:"item"`
	}

	h := fnv.New64()

	err = jsoniter.NewDecoder(io.TeeReader(res.Body, h)).Decode(&responseJson)
	if err != nil && err != io.EOF {
		d.updateErr(bgnde)
		sentry.CaptureException(err)
		return
	}

	hash := h.Sum64()
	if d.hash == hash {
		return
	}
	d.hash = hash

	// 아/점/저
	type menu struct {
		time string
		menu [5]string
	}
	var (
		menuMorning menu
		menuLunch   menu
		menuDiner   menu
	)

	for _, item := range responseJson.Item {
		var m *menu
		switch item.Type {
		case "A": // 아침
			m = &menuMorning
		case "B": // 점심
			m = &menuLunch
		case "C": // 저녁
			m = &menuDiner
		default:
			sentry.CaptureException(fmt.Errorf("unexpected value\n%+v", responseJson))
			return
		}

		m.time = item.Time
		m.menu = [5]string{item.WeekDay0, item.WeekDay1, item.WeekDay2, item.WeekDay3, item.WeekDay4}
	}

	tmplData := tmplData{
		Where:       d.Name,
		MorningTime: menuMorning.time,
		LunchTime:   menuLunch.time,
		DinerTime:   menuDiner.time,
	}

	for weekday := 0; weekday < 5; weekday++ {
		dt := bgnde.Add(time.Duration(weekday) * 24 * time.Hour)
		tmplData.Date = share.TimeFormatKr.Replace(dt.Format("2006년 1월 2일 Mon"))
		tmplData.MorningMenu = html.UnescapeString(menuMorning.menu[weekday])
		tmplData.LunchMenu = html.UnescapeString(menuLunch.menu[weekday])
		tmplData.DinerMenu = html.UnescapeString(menuDiner.menu[weekday])

		d.updateMenu(weekday, &tmplData)
	}
}

func (d *data) updateErr(bgnde time.Time) {
	tmplData := tmplData{
		Where: d.Name,
		Error: true,
	}

	for weekday := 0; weekday < 5; weekday++ {
		dt := bgnde.Add(time.Duration(weekday) * 24 * time.Hour)
		tmplData.Date = share.TimeFormatKr.Replace(dt.Format("2006년 1월 2일 Mon"))

		d.updateMenu(weekday, &tmplData)
	}
}

func (d *data) updateMenu(idx int, tmplData *tmplData) {
	sb := &d.buf
	sb.Reset()
	tmpl.Execute(sb, tmplData)

	str := strings.TrimSpace(sb.String())

	s := skill.SkillResponse{
		Version: "2.0",
		Template: skill.SkillTemplate{
			Outputs: []skill.Component{
				{
					SimpleText: &skill.SimpleText{
						Text: str,
					},
				},
			},
			QuickReplies: baseReplies,
		},
	}

	d.menu[idx].Update(share.ToBytes(str), &s)
}
