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

const (
	NoticeRange = 28 * 24 * time.Hour
	NoticeCount = 5
)

const (
	noticeTypeWeb = iota
	noticeTypeLibrary
)

type noticeInfo struct {
	once sync.Once

	Type int

	Name    string
	Url     string
	UrlView string
	Prefix  string // 전체공지에서 제목 앞에 붙여줄 내용

	skillData share.SkillData

	noticeList []noticeArticleInfo
}

type noticeArticleInfo struct {
	title       string
	postedAtStr string
	postedAt    time.Time
	id          string
	url         string
}

var (
	notice = map[int]*noticeInfo{
		공지사항: {
			Name: "공지사항 (최근 4주, 최대 5개)",
		},
		일반공지: {
			Type:    noticeTypeWeb,
			Name:    "일반공지 (최근 4주, 최대 5개)",
			Url:     "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000031/list.do",
			UrlView: "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000031/view.do?nttId=%s",
			Prefix:  "[일반] ",
		},
		학사공지: {
			Type:    noticeTypeWeb,
			Name:    "학사공지 (최근 4주, 최대 5개)",
			Url:     "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000041/list.do",
			UrlView: "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000041/view.do?nttId=%s",
			Prefix:  "[학사] ",
		},
		장학공지: {
			Type:    noticeTypeWeb,
			Name:    "장학공지 (최근 4주, 최대 5개)",
			Url:     "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000042/list.do",
			UrlView: "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000042/view.do?nttId=%s",
			Prefix:  "[장학] ",
		},
		등록공지: {
			Type:    noticeTypeWeb,
			Name:    "등록공지 (최근 4주, 최대 5개)",
			Url:     "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000052/list.do",
			UrlView: "https://www.sangji.ac.kr/prog/bbsArticle/BBSMSTR_000000000052/view.do?nttId=%s",
			Prefix:  "[등록] ",
		},
		학술공지: {
			Type:    noticeTypeLibrary,
			Name:    "학술공지 (최근 4주, 최대 5개)",
			Url:     "https://library.sangji.ac.kr/sb/default_notice_list.mir",
			UrlView: "https://library.sangji.ac.kr/sb/default_notice_view.mir?sb_no=%s",
			Prefix:  "[학술] ",
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

var searchOptionMap = map[int]struct {
	FindTr         string
	TitleTd        int
	PassAttribute  string
	NodeArticleId  func(tr, td *goquery.Selection) *goquery.Selection
	ParseArticleId *regexp.Regexp
}{
	noticeTypeWeb: {
		FindTr:         "table.board_list tbody tr",
		PassAttribute:  "notice",
		TitleTd:        1,
		NodeArticleId:  func(tr, td *goquery.Selection) *goquery.Selection { return td },
		ParseArticleId: regexp.MustCompile(`fn_search_detail\('([^']+)'`),
	},
	noticeTypeLibrary: {
		FindTr:         "table tbody tr",
		PassAttribute:  "info",
		TitleTd:        0,
		NodeArticleId:  func(tr, td *goquery.Selection) *goquery.Selection { return tr },
		ParseArticleId: regexp.MustCompile(`go_view\('([^']+)'`),
	},
}

func (n *noticeInfo) update(w *sync.WaitGroup, total bool) {
	defer w.Done()

	sortFunc := func(i, k noticeArticleInfo) bool {
		if i.postedAt.Equal(k.postedAt) {
			return i.id > k.id
		} else {
			return i.postedAt.After(k.postedAt)
		}
	}

	n.once.Do(func() {
		n.noticeList = make([]noticeArticleInfo, 5)
	})

	h := fnv.New64()
	if total {
		h.Write(notice[일반공지].skillData.GetHash())
		h.Write(notice[학사공지].skillData.GetHash())
		h.Write(notice[장학공지].skillData.GetHash())
		h.Write(notice[등록공지].skillData.GetHash())
		h.Write(notice[학술공지].skillData.GetHash())

		if !n.skillData.CheckHash(h.Sum(nil)) {
			return
		}

		noticeList := make([]noticeArticleInfo, 0, NoticeCount)

		for i, ni := range notice {
			if i == 공지사항 {
				continue
			}

			for _, nl := range ni.noticeList {
				nl.title = ni.Prefix + nl.title
				noticeList = append(noticeList, nl)
			}
		}
		sort.Slice(noticeList, func(i, k int) bool { return sortFunc(noticeList[i], noticeList[k]) })

		n.noticeList = n.noticeList[:0]
		for i := 0; i < NoticeCount && i < len(noticeList); i++ {
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

		since := time.Now().Add(-NoticeRange)

		searchOption := searchOptionMap[n.Type]

		n.noticeList = n.noticeList[:0]
		doc.Find(searchOption.FindTr).EachWithBreak(
			func(index int, tr *goquery.Selection) bool {
				if tr.HasClass(searchOption.PassAttribute) {
					return true
				}

				td := tr.Find("td")

				titleTd := td.Eq(searchOption.TitleTd).Find("a")

				// 타이틀 파싱
				title := html.UnescapeString(strings.TrimSpace(titleTd.Text()))
				if title == "" {
					return true
				}

				// 게시물 아이디 파싱
				onClick, ok := searchOption.NodeArticleId(tr, titleTd).Attr("onclick")
				if !ok {
					return true
				}
				articleIdMatches := searchOption.ParseArticleId.FindStringSubmatch(onClick)
				if articleIdMatches == nil {
					return true
				}
				articleId := articleIdMatches[1]

				// 작성일
				var postedAt time.Time
				var postedAtStr string
				td.EachWithBreak(
					func(_ int, ss *goquery.Selection) bool {
						text := html.UnescapeString(strings.TrimSpace(ss.Text()))
						t, err := time.Parse("2006-01-02", text)
						if err == nil && t.After(since) {
							postedAt = t
							postedAtStr = text
						}

						return err != nil
					},
				)
				if postedAtStr == "" {
					return true
				}

				n.noticeList = append(
					n.noticeList,
					noticeArticleInfo{
						title:       title,
						url:         fmt.Sprintf(n.UrlView, articleId),
						id:          articleId,
						postedAt:    postedAt,
						postedAtStr: postedAtStr,
					},
				)
				return len(n.noticeList) < 5
			},
		)
	}

	sort.Slice(n.noticeList, func(i, k int) bool { return sortFunc(n.noticeList[i], n.noticeList[k]) })

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
					Description: n.noticeList[k].postedAtStr,
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

	n.skillData.Update(nil, &s)
}
