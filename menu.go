package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
)

const UpdatePeriod = time.Minute * 5

type MenuData struct {
	name    string
	pageUrl string

	lock       sync.RWMutex
	menu       map[int]string // 일 : 식단
	menuBuffer [5]*strings.Builder

	updating        int32
	jsonItemCount   int
	jsonItemModDate [3]string
}

var (
	bytesPool = sync.Pool{
		New: func() interface{} {
			return new(strings.Builder)
		},
	}

	weekdayKr = []string{"월요일", "화요일", "수요일", "목요일", "일요일"}
)

var (
	MinjuStudent     = NewMenuData("https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_01/DS/getCalendar.do", "민주관 학생식당")
	MinjuProfessor   = NewMenuData("https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_02/DP/getCalendar.do", "민주관 교직원식당")
	ChangjoStudent   = NewMenuData("https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_03/CS/getCalendar.do", "창조관 학생식당")
	ChangjoProfessor = NewMenuData("https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_04/CP/getCalendar.do", "창조관 교직원식당")
)

func NewMenuData(url string, name string) MenuData {
	return MenuData{
		name:          name,
		pageUrl:       url,
		menu:          map[int]string{},
		jsonItemCount: -1,
	}
}

func MenuUpdate() {
	ticker := time.NewTicker(UpdatePeriod)

	for {
		bgnde := time.Now()

		skip := false
		switch bgnde.Weekday() {
		case time.Thursday: // 토요일 업데이트 안함
			skip = true
		case time.Sunday: // 내일 (월요일) 꺼 미리 업데이트
			bgnde.Add(24 * time.Hour)
		default:
			bgnde = bgnde.Add(time.Duration(bgnde.Weekday()-time.Monday) * -24 * time.Hour)
		}

		if !skip {
			postData := []byte(bgnde.Format("bgnde=2006-01-02"))

			go MinjuStudent.update(bgnde, postData)
			go MinjuProfessor.update(bgnde, postData)
			go ChangjoStudent.update(bgnde, postData)
			go ChangjoProfessor.update(bgnde, postData)
		}

		<-ticker.C
	}
}

func (m *MenuData) GetMenu() string {
	now := time.Now()

	weekday := now.Weekday()
	if weekday == time.Sunday || weekday == time.Saturday {
		return "주말메뉴는 제공되지 않습니다."
	}

	m.lock.RLock()
	menu, ok := m.menu[now.Day()]
	m.lock.RUnlock()

	if !ok {
		return "식단표 정보를 얻어오지 못하였습니다.\n\n잠시 후 다시 시도해주세요."
	}

	return menu
}

// with Panic
func (m *MenuData) update(bgnde time.Time, postData []byte) {
	if !atomic.CompareAndSwapInt32(&m.updating, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&m.updating, 0)

	log.Printf("Update : %s\n", m.pageUrl)

	req, _ := http.NewRequest("POST", m.pageUrl, bytes.NewReader(postData))
	req.Header = http.Header{
		"User-Agent":   []string{"sangji-haksik-channel"},
		"Content-Type": []string{"application/x-www-form-urlencoded; charset=utf-8"},
	}
	req.ContentLength = int64(len(postData))

	res, err := http.DefaultClient.Do(req)
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

	err = jsoniter.NewDecoder(res.Body).Decode(&responseJson)
	if err != nil && err != io.EOF {
		return
	}

	// 아/점/저
	var menu = make(
		[]struct {
			time string
			menu [5]string
		},
		3)

	modified := len(responseJson.Item) != m.jsonItemCount

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
			sentry.CaptureException(errors.New(fmt.Sprintf("Unexpected Value\n%+v", responseJson)))
			return
		}

		menu[index].time = item.Time
		menu[index].menu = [5]string{item.WeekDay0, item.WeekDay1, item.WeekDay2, item.WeekDay3, item.WeekDay4}

		modified = modified || m.jsonItemModDate[index] != item.ModDate
		jsonItemModDate[index] = item.ModDate
	}

	if !modified {
		return
	}

	m.lock.Lock()
	defer m.lock.Unlock()

	m.jsonItemCount = len(responseJson.Item)
	m.jsonItemModDate = jsonItemModDate

	// clear
	for key := range m.menu {
		delete(m.menu, key)
	}
	for i := 0; i < 5; i++ {
		menu[0].menu[i] = strings.ReplaceAll(menu[0].menu[i], "&amp;amp;", "&")
		menu[1].menu[i] = strings.ReplaceAll(menu[1].menu[i], "&amp;amp;", "&")
		menu[2].menu[i] = strings.ReplaceAll(menu[2].menu[i], "&amp;amp;", "&")

		/**
		2020년 02월 25일 토요일
		민주관 학생식당

		----------------------
		아침 (09:00 ~ 10:00)

		북어해장국
		공기밥
		깍두기

		----------------------
		점심 (11:00 ~ 14:00)

		일품:돈가스카레덮밥/쥬시쿨
		백반:돈육바베큐볶음
		미역국
		계란찜
		파래김자반

		----------------------
		저녁 (17:00 ~ 18:30)

		일품:돈가스카레덮밥/쥬시쿨
		백반:돈육바베큐볶음
		미역국
		계란찜
		파래김자반
		*/
		sb := bytesPool.Get().(*strings.Builder)

		dt := bgnde.Add(time.Duration(i) * 24 * time.Hour)
		fmt.Fprintln(sb, dt.Format("2006년 01월 02일"), weekdayKr[dt.Weekday()-1])
		fmt.Fprintln(sb, m.name)
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

		if m.menuBuffer[i] != nil {
			m.menuBuffer[i].Reset()
			bytesPool.Put(m.menuBuffer[i])
		}
		m.menu[dt.Day()] = sb.String()
		m.menuBuffer[i] = sb
	}
}
