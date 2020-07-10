package srvnotice

import (
	"fmt"
	"hash/fnv"
	"html"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"sangjihaksik/share"

	"github.com/PuerkitoBio/goquery"
	skill "github.com/RyuaNerin/go-kakaoskill/v2"
	"github.com/getsentry/sentry-go"
)

type noticeInfo struct {
	once sync.Once

	Name    string
	Url     string
	UrlView string

	skillData share.SkillData

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
			Name: "공지사항 (최근 4주, 최대 15개)",
		},
		일반공지: {
			Name:    "일반공지 (최근 4주, 최대 5개)",
			Url:     "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000031/list.do",
			UrlView: "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000031/view.do?nttId=%s",
		},
		학사공지: {
			Name:    "학사공지 (최근 4주, 최대 5개)",
			Url:     "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000041/list.do",
			UrlView: "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000041/view.do?nttId=%s",
		},
		장학공지: {
			Name:    "장학공지 (최근 4주, 최대 5개)",
			Url:     "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000042/list.do",
			UrlView: "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000042/view.do?nttId=%s",
		},
		등록공지: {
			Name:    "등록공지 (최근 4주, 최대 5개)",
			Url:     "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000052/list.do",
			UrlView: "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000052/view.do?nttId=%s",
		},
	}
)

func init() {
	share.DoUpdate(share.Config.UpdatePeriodNotice, update)
}

func update() {
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
		n.noticeList = make([]noticeArticleInfo, 5)
	})

	h := fnv.New64()
	if total {
		h.Write(notice[일반공지].skillData.GetHash())
		h.Write(notice[학사공지].skillData.GetHash())
		h.Write(notice[장학공지].skillData.GetHash())
		h.Write(notice[등록공지].skillData.GetHash())

		if !n.skillData.CheckHash(h.Sum(nil)) {
			return
		}

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
				return noticeList[i].id < noticeList[k].id
			},
		)

		n.noticeList = n.noticeList[:0]
		for i := 0; i < 15 && i < len(noticeList); i++ {
			n.noticeList = append(n.noticeList, noticeList[i])
		}
	} else {
		if n.Url == "" {
			return
		}

		req, _ := http.NewRequest("GET", n.Url, nil)
		req.Header = http.Header{
			"User-Agent": []string{share.UserAgent},
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			sentry.CaptureException(err)
			return
		}
		defer res.Body.Close()

		doc, err := goquery.NewDocumentFromReader(io.TeeReader(res.Body, h))
		if err != nil && err != io.EOF {
			sentry.CaptureException(err)
			return
		}

		if !n.skillData.CheckHash(h.Sum(nil)) {
			return
		}

		since := time.Now().Add(-share.Config.NoticeRange)

		n.noticeList = n.noticeList[:0]
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
						t, err := time.Parse("2006-01-02", text)
						if err == nil && t.After(since) {
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
				return len(n.noticeList) < 5
			},
		)
	}

	sort.Slice(
		n.noticeList,
		func(i, k int) bool {
			return n.noticeList[i].id < n.noticeList[k].id
		},
	)

	s := skill.SkillResponse{
		Version: "2.0",
		Template: skill.SkillTemplate{
			Outputs:      make([]skill.Component, 0, 3),
			QuickReplies: baseReplies,
		},
	}

	for i := 0; i < len(n.noticeList); i += 5 {
		items := make([]skill.ListItemItems, 0, 5)

		for k := i; k < i+5 && k < len(n.noticeList); k++ {
			items = append(
				items,
				skill.ListItemItems{
					Title:       n.noticeList[k].title,
					Description: n.noticeList[k].postedAt,
					Link: skill.Link{
						Web: n.noticeList[k].url,
					},
				},
			)
		}

		s.Template.Outputs = append(
			s.Template.Outputs,
			skill.Component{
				ListCard: &skill.ListCard{
					Header: skill.ListItemHeader{
						Title: n.Name,
						Link: skill.Link{
							Web: n.Url,
						},
					},
					Items: items,
				},
			},
		)
	}

	n.skillData.Update(&s)
}
