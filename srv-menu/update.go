package srvmenu

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"html"
	"io"
	"net/http"
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

	// 월 ~ 금
	menu [5]share.SkillData

	// 일일 시간표 텍스트 생성에 사용될 버퍼
	menuStringBuffer bytes.Buffer

	// 메뉴 업데이트 확인용
	hash uint64
}

var (
	menu = map[int]*data{
		민주학생: {
			Name: "민주관 학생식당",
			Url:  "https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_01/DS/getCalendar.do",
		},
		민주교직: {
			Name: "민주관 교직원식당",
			Url:  "https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_02/DP/getCalendar.do",
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
		sentry.CaptureException(err)
		return
	}

	hash := h.Sum64()
	if d.hash == hash {
		return
	}
	d.hash = hash

	// 아/점/저
	var menu = make(
		[]struct {
			time string
			menu [5]string
		},
		3)

	var jsonItemModDate [3]string

	for _, item := range responseJson.Item {
		index := 0
		switch item.Type {
		case "A": // 아침
			index = 0
		case "B": // 점심
			index = 1
		case "C": // 저녁
			index = 2
		default:
			sentry.CaptureException(fmt.Errorf("unexpected value\n%+v", responseJson))
			return
		}

		menu[index].time = item.Time
		menu[index].menu = [5]string{item.WeekDay0, item.WeekDay1, item.WeekDay2, item.WeekDay3, item.WeekDay4}

		jsonItemModDate[index] = item.ModDate
	}

	sb := &d.menuStringBuffer

	for i := 0; i < 5; i++ {
		menu[0].menu[i] = html.UnescapeString(menu[0].menu[i])
		menu[1].menu[i] = html.UnescapeString(menu[1].menu[i])
		menu[2].menu[i] = html.UnescapeString(menu[2].menu[i])

		/**
		2020년 2월 2일 토요일
		민주관 학생식당

		----------------------
		아침 (09:00 ~ 10:00)
		북어해장국
		공기밥
		깍두기
		----------------------
		점심 (11:00 ~ 14:00)
		메뉴없음
		----------------------
		저녁 (17:00 ~ 18:30)
		일품:돈가스카레덮밥/쥬시쿨
		백반:돈육바베큐볶음
		미역국
		계란찜
		파래김자반
		*/
		sb.Reset()

		dt := bgnde.Add(time.Duration(i) * 24 * time.Hour)
		fmt.Fprintln(sb, share.TimeFormatKr.Replace(dt.Format("2006년 1월 2일 Mon")))
		fmt.Fprintln(sb, d.Name)
		fmt.Fprintln(sb)

		// 메뉴 없음
		if len(menu[0].menu[i]) == 0 && len(menu[1].menu[i]) == 0 && len(menu[2].menu[i]) == 0 {
			fmt.Fprint(sb, "메뉴 없음")
		} else {
			fmt.Fprintln(sb, "---------------------")

			if len(menu[0].menu[i]) > 0 {
				fmt.Fprintf(sb, "아침 (%s)", menu[0].time)
				fmt.Fprintln(sb)
				fmt.Fprintln(sb, menu[0].menu[i])
			} else {
				fmt.Fprintln(sb, "아침")
				fmt.Fprintln(sb, "메뉴 없음")
			}

			fmt.Fprintln(sb, "---------------------")

			if len(menu[1].menu[i]) > 0 {
				fmt.Fprintf(sb, "점심 (%s)", menu[1].time)
				fmt.Fprintln(sb)
				fmt.Fprintln(sb, menu[1].menu[i])
			} else {
				fmt.Fprintln(sb, "점심")
				fmt.Fprintln(sb, "메뉴 없음")
			}

			fmt.Fprintln(sb, "---------------------")

			if len(menu[2].menu[i]) > 0 {
				fmt.Fprintf(sb, "저녁 (%s)", menu[2].time)
				fmt.Fprintln(sb)
				fmt.Fprint(sb, menu[2].menu[i])
			} else {
				fmt.Fprintln(sb, "저녁")
				fmt.Fprint(sb, "메뉴 없음")
			}
		}

		s := skill.SkillResponse{
			Version: "2.0",
			Template: skill.SkillTemplate{
				Outputs: []skill.Component{
					{
						SimpleText: &skill.SimpleText{
							Text: share.ToString(sb.Bytes()),
						},
					},
				},
				QuickReplies: baseReplies,
			},
		}

		d.menu[i].Update(sb.Bytes(), &s)
	}
}
