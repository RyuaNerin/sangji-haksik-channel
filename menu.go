package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
)

type MenuData struct {
	MinjuStudent     weeklyMenu // 민주관 학생
	MinjuProfessor   weeklyMenu // 민주관 교직원
	ChangjoStudent   weeklyMenu // 창조관 학생
	ChangjoProfessor weeklyMenu // 창조관 교직원
}

type weeklyMenu struct {
	morningTime string
	morning     [5]string // 아침 : 0월 1화 2수 3목 4금
	lunchTime   string
	lunch       [5]string // 점심 : 0월 1화 2수 3목 4금
	dinnerTime  string
	dinner      [5]string // 저녁 : 0월 1화 2수 3목 4금

	dailyMenu [5]string // 일일 식단표
}

var (
	menuLock  sync.Mutex
	menuBgnde time.Time
	menuCache *MenuData
)

// nil, true = 메뉴 제공 안함
func getMenu() (menu *MenuData, weekday int, ok bool) {
	menuLock.Lock()
	defer menuLock.Unlock()

	bgnde := time.Now()
	switch bgnde.Weekday() {
	case time.Monday:
		weekday = 0
	case time.Tuesday:
		weekday = 1
		bgnde = bgnde.Add(-1 * 24 * time.Hour)
	case time.Wednesday:
		weekday = 2
		bgnde = bgnde.Add(-2 * 24 * time.Hour)
	case time.Thursday:
		weekday = 3
		bgnde = bgnde.Add(-3 * 24 * time.Hour)
	case time.Friday:
		weekday = 4
		bgnde = bgnde.Add(-4 * 24 * time.Hour)
	case time.Saturday:
	case time.Sunday:
		return nil, -1, true
	}

	uy, um, ud := menuBgnde.Date()
	ny, nm, nd := bgnde.Date()

	if uy == ny && um == nm && ud == nd {
		return menuCache, weekday, true
	}

	postData := []byte(fmt.Sprintf("bgnde=%4d-%02d-%02d", ny, nm, nd))

	ms, ok := updateMenu("https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_01/DS/getCalendar.do", postData, bgnde, "민주관 학생식당")
	if !ok {
		return nil, -1, false
	}
	mp, ok := updateMenu("https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_02/DP/getCalendar.do", postData, bgnde, "민주관 교직원식당")
	if !ok {
		return nil, -1, false
	}
	cs, ok := updateMenu("https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_03/CS/getCalendar.do", postData, bgnde, "창조관 학생식당")
	if !ok {
		return nil, -1, false
	}
	cp, ok := updateMenu("https://www.sangji.ac.kr/prog/carteGuidance/kor/sub07_10_04/CP/getCalendar.do", postData, bgnde, "창조관 교직원식당")
	if !ok {
		return nil, -1, false
	}

	menuBgnde = bgnde
	menuCache = &MenuData{
		MinjuStudent:     ms,
		MinjuProfessor:   mp,
		ChangjoStudent:   cs,
		ChangjoProfessor: cp,
	}

	return menuCache, weekday, true
}

// with Panic
func updateMenu(url string, postData []byte, bgnde time.Time, name string) (menu weeklyMenu, ok bool) {
	log.Printf("Update : %s\n", url)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(postData))
	req.Header = http.Header{
		"User-Agent":   []string{"sangji-haksik-channel"},
		"Content-Type": []string{"application/x-www-form-urlencoded; charset=utf-8"},
		"Referer":      []string{url},
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

	for _, item := range responseJson.Item {
		switch item.Type {
		case "A": // 아침
			menu.morningTime = item.Time
			menu.morning = [5]string{
				item.WeekDay0,
				item.WeekDay1,
				item.WeekDay2,
				item.WeekDay3,
				item.WeekDay4,
			}

		case "B": // 점심
			menu.lunchTime = item.Time
			menu.lunch = [5]string{
				item.WeekDay0,
				item.WeekDay1,
				item.WeekDay2,
				item.WeekDay3,
				item.WeekDay4,
			}

		case "C": // 저녁
			menu.dinnerTime = item.Time
			menu.dinner = [5]string{
				item.WeekDay0,
				item.WeekDay1,
				item.WeekDay2,
				item.WeekDay3,
				item.WeekDay4,
			}
		}
	}

	dateString := bgnde.Format("2006년 01월 02일")

	for i := 0; i < 5; i++ {
		menu.morning[i] = strings.ReplaceAll(menu.morning[i], "&amp;amp;", "&")
		menu.lunch[i] = strings.ReplaceAll(menu.lunch[i], "&amp;amp;", "&")
		menu.dinner[i] = strings.ReplaceAll(menu.dinner[i], "&amp;amp;", "&")

		// 일일 전체 식단표 문자열 사전생성 하는 부분
		/**
		2020년 02월 25일
		민주관 학생식당

		아침
		09:00 ~ 10:00

		북어해장국
		공기밥
		깍두기

		-----------
		점심
		11:00 ~ 14:00

		일품:돈가스카레덮밥/쥬시쿨
		백반:돈육바베큐볶음
		미역국
		계란찜
		파래김자반

		-----------
		저녁
		17:00 ~ 18:30

		일품:돈가스카레덮밥/쥬시쿨
		백반:돈육바베큐볶음
		미역국
		계란찜
		파래김자반
		*/
		var sb strings.Builder
		fmt.Fprintln(&sb, dateString)
		fmt.Fprintln(&sb, name)
		fmt.Fprintln(&sb)

		bef := false

		if len(menu.morning[i]) > 0 {
			fmt.Fprintln(&sb, "아침")
			fmt.Fprintln(&sb, menu.morningTime)
			fmt.Fprintln(&sb)
			fmt.Fprint(&sb, menu.morning[i])

			bef = true
		}

		if len(menu.lunch[i]) > 0 {
			if bef {
				fmt.Fprintln(&sb)
				fmt.Fprintln(&sb)
				fmt.Fprintln(&sb, "----------")
			}

			fmt.Fprintln(&sb, "점심")
			fmt.Fprintln(&sb, menu.lunchTime)
			fmt.Fprintln(&sb)
			fmt.Fprint(&sb, menu.lunch[i])

			bef = true
		}

		if len(menu.dinner[i]) > 0 {
			if bef {
				fmt.Fprintln(&sb)
				fmt.Fprintln(&sb)
				fmt.Fprintln(&sb, "----------")
			}

			fmt.Fprintln(&sb, "저녁")
			fmt.Fprintln(&sb, menu.dinnerTime)
			fmt.Fprintln(&sb)
			fmt.Fprint(&sb, menu.dinner[i])

			bef = true
		}

		if !bef {
			fmt.Fprint(&sb, "메뉴 없음")
		}

		menu.dailyMenu[i] = sb.String()

		/**
		2020년 02월 25일
		민주관 학생식당 - 점심
		09:00 ~ 10:00

		북어해장국
		공기밥
		깍두기
		*/

		fm := func(tag string, menu string, time string) string {
			if len(menu) > 0 {
				return fmt.Sprintf("%s\n%s - %s\n%s\n\n%s", dateString, name, tag, time, menu)
			} else {
				return fmt.Sprintf("%s\n%s - %s\n\n메뉴 없음", dateString, name, tag)
			}
		}

		menu.morning[i] = fm("아침", menu.morning[i], menu.morningTime)
		menu.lunch[i] = fm("점심", menu.lunch[i], menu.lunchTime)
		menu.dinner[i] = fm("저녁", menu.dinner[i], menu.dinnerTime)
	}

	return menu, true
}
