package srvbus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"sangjihaksik/share"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
)

type stationInfo struct {
	StationName string // 정류장 이름
	RequestBody []byte // api 호출용

	arrivalList []arrivalInfo
	updateError bool
}
type arrivalInfo struct {
	number         string
	remainMinutes  int // 몇 분 남음
	remainStations int // 몇 정거장 남음
}

type routeInfo struct {
	Name        string           // 목적지 이름
	StationList map[int][]string // 정류장 목록
	MemoBus     []string         // 메모 표시할 버스
	Memo        string           // 메모

	busArrivalListTemp []arrivalInfo

	skillData share.SkillData

	bodyBuffer bytes.Buffer // 텍스트 버퍼
}

var (
	stationList = map[int]*stationInfo{
		우산초교: {
			StationName: "우산초교 (정문)",
			RequestBody: []byte("stationId=251061041"),
		},
		강원정비기술학원: {
			StationName: "강원정비기술학원 (상지마트)",
			RequestBody: []byte("stationId=251061013"),
		},
		터미널앞: {
			StationName: "터미널 앞",
			RequestBody: []byte("stationId=251060037"),
		},
		터미널맞은편: {
			StationName: "터미널 길건너",
			RequestBody: []byte("stationId=251060036"),
		},
		원주역: {
			StationName: "원주역 (CU 앞)",
			RequestBody: []byte("stationId=251058010"),
		},
	}

	routeList = map[int]*routeInfo{
		SchoolToTerminal: {
			Name: "학교 > 터미널",
			StationList: map[int][]string{
				우산초교: {
					"4",
					"13",
					"30",
				},
				강원정비기술학원: {
					"2-1",
					"16",
					"31",
					"34",
					"90",
				},
			},
		},
		TerminalToSchool: {
			Name: "터미널 > 학교",
			StationList: map[int][]string{
				터미널앞: {
					"2-1",
					"4",
					"16",
					"30",
					"31",
					"34",
					"90",
				},
				터미널맞은편: {
					"13",
				},
			},
			Memo: "우산초교 (정문)",
			MemoBus: []string{
				"4",
				"13",
				"30",
			},
		},
		SchoolToStation: {
			Name: "학교 > 원주역",
			StationList: map[int][]string{
				강원정비기술학원: {
					"2",
					"16-1",
					"21", "22", "23", "24",
					"32",
					"41", "41-2",
					"82",
					"90",
				},
			},
		},
		StationToSchool: {
			Name: "원주역 > 학교",
			StationList: map[int][]string{
				원주역: {
					"2",
					"16-1",
					"21", "22", "23", "24",
					"32",
					"41", "41-2",
					"82",
					"90",
				},
			},
		},
	}

	httpClient = share.NewHttpClient()
)

func init() {
	share.DoUpdate(share.Config.UpdatePeriodBus.Value(), update)
}

func update() {
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

	req, _ := http.NewRequest(http.MethodPost, "http://its.wonju.go.kr/bus/busStaionAjax.do", bytes.NewReader(si.RequestBody))
	req.Header = http.Header{
		"User-Agent":   {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36"},
		"Content-Type": {"application/x-www-form-urlencoded; charset=UTF-8"},
		"Origin":       {"http://its.wonju.go.kr"},
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	var busStationAjax struct {
		Bus []struct {
			PredictTm     int    `json:"predictTm,omitempty"`
			ProvideType   int    `json:"provideType,omitempty"`
			RemainStation int    `json:"remainStation,omitempty"`
			RouteNm       string `json:"routeNm"`
		} `json:"bus"`
	}

	err = json.NewDecoder(res.Body).Decode(&busStationAjax)
	if err != nil && err != io.EOF {
		si.updateError = true
		return
	}
	si.updateError = false

	for _, bus := range busStationAjax.Bus {
		if bus.ProvideType != 1 {
			continue
		}

		si.arrivalList = append(
			si.arrivalList,
			arrivalInfo{
				number:         bus.RouteNm,
				remainMinutes:  bus.PredictTm,
				remainStations: bus.RemainStation,
			},
		)
	}
}

func (ri *routeInfo) update(w *sync.WaitGroup) {
	defer w.Done()

	/**
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
		if si.updateError {
			fmt.Fprintf(&ri.bodyBuffer, "\n- %s\n정보 조회 오류\n", si.StationName)
			bodyWritten = true
			continue
		}
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
	sr := skill.SkillResponse{
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

	ri.skillData.Update(nil, &sr)
}
