package srvnotice

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"sort"
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

type noticeInfo struct {
	once sync.Once

	Name    string
	Url     string
	UrlView string

	skillResponseLock       sync.RWMutex
	skillResponseData       []byte
	skillResponseDataBuffer bytes.Buffer

	skillResponseStruct skill.SkillResponse

	noticeList []noticeArticleInfo
}

type noticeArticleInfo struct {
	title    string
	postedAt string
	id       string
	url      string
}

var (
	notice = map[int]*noticeInfo{
		공지사항: {
			Name: "공지사항",
		},
		일반공지: {
			Name:    "일반공지",
			Url:     "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000031/list.do",
			UrlView: "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000031/view.do?nttId=%s",
		},
		학사공지: {
			Name:    "학사공지",
			Url:     "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000041/list.do",
			UrlView: "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000041/view.do?nttId=%s",
		},
		장학공지: {
			Name:    "장학공지",
			Url:     "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000042/list.do",
			UrlView: "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000042/view.do?nttId=%s",
		},
		등록공지: {
			Name:    "등록공지",
			Url:     "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000052/list.do",
			UrlView: "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000052/view.do?nttId=%s",
		},
	}

	baseReplies = []skill.QuickReply{
		{
			Label:       "공지",
			Action:      "message",
			MessageText: "공지사항",
		},
		{
			Label:       "일반",
			Action:      "message",
			MessageText: "일반공지",
		},
		{
			Label:       "학사",
			Action:      "message",
			MessageText: "학사공지",
		},
		{
			Label:       "장학",
			Action:      "message",
			MessageText: "장학공지",
		},
		{
			Label:       "등록",
			Action:      "message",
			MessageText: "등록공지",
		},
	}
)

func init() {
	go func() {
		ticker := time.NewTicker(share.Config.UpdatePeriodNotice)

		update()

		<-ticker.C
	}()
}

var updateIsRunning int32 = 0

func update() {
	if atomic.SwapInt32(&updateIsRunning, 1) != 0 {
		return
	}
	defer atomic.StoreInt32(&updateIsRunning, 0)

	var w sync.WaitGroup
	for _, n := range notice {
		w.Add(1)
		go n.update(&w, false)
	}

	w.Wait()

	w.Add(1)
	notice[공지사항].update(&w, true)
}

var regexArticleId = regexp.MustCompile(`fn_search_detail\('([^']+)'\)`)

func (n *noticeInfo) update(w *sync.WaitGroup, total bool) {
	defer w.Done()

	n.once.Do(func() {
		n.skillResponseStruct = skill.SkillResponse{
			Version: "2.0",
			Template: skill.SkillTemplate{
				Outputs: []skill.Component{
					{
						ListCard: &skill.ListCard{
							Header: skill.ListItemHeader{
								Title: n.Name,
								Link: skill.Link{
									Web: n.Url,
								},
							},
							Items: make([]skill.ListItemItems, 0, 5),
						},
					},
				},
				QuickReplies: baseReplies,
			},
		}
	})

	n.noticeList = n.noticeList[:0]

	if total {
		var noticeList []noticeArticleInfo

		for i, ni := range notice {
			if i == 공지사항 {
				continue
			}
			noticeList = append(noticeList, ni.noticeList...)
		}
		sort.Slice(
			noticeList,
			func(i, k int) bool {
				return noticeList[i].id > noticeList[k].id
			},
		)

		for i := 0; i < 5; i++ {
			n.noticeList = append(n.noticeList, noticeList[i])
		}
	} else {
		req, _ := http.NewRequest("GET", n.Url, nil)
		req.Header = http.Header{
			"User-Agent": []string{share.UserAgent},
		}
		res, err := share.Client.Do(req)
		if err != nil {
			sentry.CaptureException(err)
			return
		}
		defer res.Body.Close()

		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil && err != io.EOF {
			sentry.CaptureException(err)
			return
		}

		doc.Find("table.board_list tr").EachWithBreak(
			func(index int, s *goquery.Selection) bool {
				if s.HasClass("notice") {
					return true
				}

				td := s.Find("td")

				titleTd := td.Eq(1).Find("a")

				// 타이틀 파싱
				title := html.UnescapeString(strings.TrimSpace(titleTd.Text()))
				if title == "" {
					return true
				}

				// 게시물 아이디 파싱
				onClick, ok := titleTd.Attr("onclick")
				if !ok {
					return true
				}
				id := regexArticleId.FindStringSubmatch(onClick)
				if id == nil {
					return true
				}

				// 작성일
				var postedAt string
				td.Each(
					func(_ int, ss *goquery.Selection) {
						text := html.UnescapeString(strings.TrimSpace(ss.Text()))
						_, err := time.Parse("2006-01-02", text)
						if err == nil {
							postedAt = text
						}
					},
				)
				if postedAt == "" {
					return true
				}

				n.noticeList = append(
					n.noticeList,
					noticeArticleInfo{
						title:    title,
						url:      fmt.Sprintf(n.UrlView, id[1]),
						id:       id[1],
						postedAt: postedAt,
					},
				)
				if len(n.noticeList) == 5 {
					return false
				}

				return true
			},
		)
	}

	n.skillResponseStruct.Template.Outputs[0].ListCard.Items = n.skillResponseStruct.Template.Outputs[0].ListCard.Items[:0]
	for _, ni := range n.noticeList {
		n.skillResponseStruct.Template.Outputs[0].ListCard.Items = append(
			n.skillResponseStruct.Template.Outputs[0].ListCard.Items,
			skill.ListItemItems{
				Title:       ni.title,
				Description: ni.postedAt,
				Link: skill.Link{
					Web: ni.url,
				},
			},
		)
	}

	n.skillResponseLock.Lock()
	defer n.skillResponseLock.Unlock()

	n.skillResponseData = nil

	n.skillResponseDataBuffer.Reset()
	err := jsoniter.NewEncoder(&n.skillResponseDataBuffer).Encode(&n.skillResponseStruct)
	if err != nil {
		sentry.CaptureException(err)
		return
	}

	n.skillResponseData = n.skillResponseDataBuffer.Bytes()
}
