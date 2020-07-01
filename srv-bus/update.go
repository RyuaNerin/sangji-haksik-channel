package srvbus

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sangjihaksik/share"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	skill "github.com/RyuaNerin/go-kakaoskill/v2"
	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
)

/*
[학교 > 터미널]

- 우산초교
30 : 2분 후 (2 정거장 전)

- 상지마트 앞
도착 예정 버스 없음

----------------------------------------

[터미널 > 학교]

- 터미널 앞
2-1 : 3분 후 (2 정거장 전) *
4 : 3분 후 (2 정거장 전) *

- 터미널 건너편
도착 예정 버스 없음

* 학교 정문
*/

var (
	baseReplies = []skill.QuickReply{
		{
			Label:       "학→터",
			Action:      "message",
			MessageText: "학→터",
		},
		{
			Label:       "터→학",
			Action:      "message",
			MessageText: "터→학",
		},
		{
			Label:       "학→원",
			Action:      "message",
			MessageText: "학→원",
		},
		{
			Label:       "원→학",
			Action:      "message",
			MessageText: "원→학",
		},
	}
)

func init() {
	go updateFunc()
}

func updateFunc() {
	ticker := time.NewTicker(share.Config.UpdatePeriodBus)

	for {
		go update()

		<-ticker.C
	}
}

var uploadLock int32

func update() {
	if atomic.SwapInt32(&uploadLock, 1) != 0 {
		return
	}
	defer atomic.StoreInt32(&uploadLock, 0)

	var w sync.WaitGroup

	for _, si := range stationList {
		w.Add(1)
		go si.update(&w)
	}
	w.Wait()

	for _, ri := range routeList {
		w.Add(1)
		go ri.update(&w)
	}
	w.Wait()
}

func (si *stationInfo) update(w *sync.WaitGroup) {
	defer w.Done()

	si.arrivalList = si.arrivalList[:0]

	req, _ := http.NewRequest("POST", "http://its.wonju.go.kr:8090/map/AjaxRouteListByStop.do", bytes.NewReader(si.RequestBody))
	req.Header = http.Header{
		"User-Agent":   {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36"},
		"Content-Type": {"application/x-www-form-urlencoded; charset=UTF-8"},
	}

	res, err := share.Client.Do(req)
	if err != nil {
		sentry.CaptureException(err)
		return
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil && err != io.EOF {
		sentry.CaptureException(err)
		return
	}

	doc.Find("div.nw_table1 tr[id]").Each(
		func(index int, s *goquery.Selection) {
			busName := strings.TrimSpace(s.Find("th p").First().Text())
			if busName == "" {
				return
			}

			ss := s.Find("td font")
			remainStation, err := strconv.Atoi(strings.TrimSpace(ss.Eq(0).Text()))
			if err != nil {
				return
			}

			remainMinutes, err := strconv.Atoi(strings.TrimSpace(ss.Eq(1).Text()))
			if err != nil {
				return
			}

			si.arrivalList = append(
				si.arrivalList,
				arrivalInfo{
					number:         busName,
					remainMinutes:  remainMinutes,
					remainStations: remainStation,
				},
			)
		},
	)
}

func (ri *routeInfo) update(w *sync.WaitGroup) {
	defer w.Done()

	ri.bodyBuffer.Reset()

	now := time.Now()
	fmt.Fprintf(
		&ri.bodyBuffer,
		"[%s]\n%d시 %d분 %d초 기준\n",
		ri.Name,
		now.Hour(),
		now.Minute(),
		now.Second(),
	)

	bodyWritten := false
	memoWritten := false

	for sk, busList := range ri.StationList {
		si := stationList[sk]
		if si.arrivalList == nil {
			continue
		}

		ri.busArrivalListTemp = ri.busArrivalListTemp[:0]
		for _, busNumber := range busList {
			for _, arrival := range si.arrivalList {
				if arrival.number != busNumber {
					continue
				}

				ri.busArrivalListTemp = append(ri.busArrivalListTemp, arrival)
				break
			}
		}

		if len(ri.busArrivalListTemp) > 0 {
			sort.Slice(
				ri.busArrivalListTemp,
				func(i, j int) bool {
					if ri.busArrivalListTemp[i].remainMinutes == ri.busArrivalListTemp[j].remainMinutes {
						return ri.busArrivalListTemp[i].remainStations <= ri.busArrivalListTemp[j].remainStations
					}

					return ri.busArrivalListTemp[i].remainMinutes <= ri.busArrivalListTemp[j].remainMinutes
				},
			)

			bodyWritten = true
			fmt.Fprintf(&ri.bodyBuffer, "\n- %s\n", si.StationName)

			for _, arrival := range ri.busArrivalListTemp {
				withMemo := false
				for _, busNumberWithMemo := range ri.MemoBus {
					if arrival.number == busNumberWithMemo {
						withMemo = true
						memoWritten = true
						break
					}
				}

				if withMemo {
					fmt.Fprintf(&ri.bodyBuffer, "%s : %d 분 전 (%d 정거장 전) *\n", arrival.number, arrival.remainMinutes, arrival.remainStations)
				} else {
					fmt.Fprintf(&ri.bodyBuffer, "%s : %d 분 전 (%d 정거장 전)\n", arrival.number, arrival.remainMinutes, arrival.remainStations)
				}
			}
		}
	}

	if !bodyWritten {
		fmt.Fprint(&ri.bodyBuffer, "\n도착 예정 버스 없음")
	} else {
		if memoWritten {
			fmt.Fprintf(&ri.bodyBuffer, "\n* %s", ri.Memo)
		}
	}

	body := strings.TrimSpace(share.ToString(ri.bodyBuffer.Bytes()))
	skillResponse := skill.SkillResponse{
		Version: "2.0",
		Template: skill.SkillTemplate{
			Outputs: []skill.Component{
				{
					SimpleText: &skill.SimpleText{
						Text: body,
					},
				},
			},
			QuickReplies: baseReplies,
		},
	}

	ri.skillResponseBodyBuffer.Reset()
	err := jsoniter.NewEncoder(&ri.skillResponseBodyBuffer).Encode(&skillResponse)
	if err != nil {
		sentry.CaptureException(err)
		return
	}

	ri.skillResponseLock.Lock()
	defer ri.skillResponseLock.Unlock()
	ri.skillResponseBody = ri.skillResponseBodyBuffer.Bytes()
}
