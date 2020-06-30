package srvbus

import (
	"bytes"
	"sync"
)

type stationInfo struct {
	StationName string // 정류장 이름
	RequestBody []byte // api 호출용

	arrivalList []arrivalInfo
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

	skillResponseLock       sync.RWMutex
	skillResponseBody       []byte       // 스킬 응답 사전 생성
	skillResponseBodyBuffer bytes.Buffer // skillResponse to byte 버퍼

	bodyBuffer bytes.Buffer // 텍스트 버퍼
}

const (
	_ = iota
	우산초교
	강원정비기술학원
	터미널앞
	터미널맞은편
	원주역
)

const (
	SchoolToTerminal = "1"
	TerminalToSchool = "2"
	SchoolToStation  = "3"
	StationToSchool  = "4"
)

var (
	stationList = map[int]*stationInfo{
		우산초교: {
			StationName: "우산초교 (정문)",
			RequestBody: []byte("station_id=251061041"),
		},
		강원정비기술학원: {
			StationName: "강원정비기술학원 (상지마트)",
			RequestBody: []byte("station_id=251061013"),
		},
		터미널앞: {
			StationName: "터미널 앞",
			RequestBody: []byte("station_id=251060037"),
		},
		터미널맞은편: {
			StationName: "터미널 길건너",
			RequestBody: []byte("station_id=251060036"),
		},
		원주역: {
			StationName: "원주역 (CU 앞)",
			RequestBody: []byte("station_id=251058010"),
		},
	}

	routeList = map[string]*routeInfo{
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
			Memo: "우산초교 (정문)",
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
)
