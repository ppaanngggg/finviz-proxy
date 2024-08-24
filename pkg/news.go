package pkg

import (
	"bytes"
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func fetchAllNews(ctx context.Context, isElite bool) ([]byte, error) {
	url := "https://finviz.com/news.ashx"
	if isElite {
		url = "https://elite.finviz.com/news.ashx"
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		slog.Error("fetchAllNews http new request", "err", err)
		return nil, err
	}
	req.Header.Set("User-Agent", "curl/7.88.1")
	client := newClient()
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("fetchAllNews http do", "err", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Error("fetchAllNews status code not ok", "code", resp.StatusCode)
		return nil, errors.New("fetchAllNews status code not ok")
	}
	page, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("fetchAllNews read all http resp body", "err", err)
		return nil, err
	}
	return page, nil
}

type Record struct {
	Date  string `json:"date"` // Jan-02 2006
	Title string `json:"title"`
	URL   string `json:"url"`
}

func parseLinks(table *goquery.Selection) []Record {
	loc, _ := time.LoadLocation("America/New_York")
	today := time.Now().UTC().In(loc)
	var records []Record
	table.Find("tr.news_table-row").Each(func(i int, tr *goquery.Selection) {
		a := tr.Find("a")
		href, exists := a.Attr("href")
		if !exists {
			return
		}
		date := strings.TrimSpace(tr.Find("td.news_date-cell").Text())
		if strings.HasSuffix(date, "AM") || strings.HasSuffix(date, "PM") {
			// if 05:30AM, format today
			date = today.Format("Jan-02 2006")
		} else {
			date += " " + today.Format("2006") // add year
		}
		text := a.Text()
		records = append(records, Record{
			Date:  date,
			Title: text,
			URL:   href,
		})
	})
	return records
}

func parseNewsAndBlogs(page []byte) ([]Record, []Record, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if err != nil {
		slog.Error("failed to parse news and blogs from page", "err", err)
		return nil, nil, err
	}
	var newsTable *goquery.Selection
	var blogsTable *goquery.Selection
	doc.Find("table.styled-table-new").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			newsTable = s
		}
		if i == 1 {
			blogsTable = s
		}
	})
	return parseLinks(newsTable), parseLinks(blogsTable), nil
}

func FetchAndParseNewsAndBlogs(ctx context.Context, isElite bool) ([]Record, []Record, error) {
	// fetch page
	page, err := fetchAllNews(ctx, isElite)
	if err != nil {
		slog.Error("failed to fetch news and blogs", "err", err)
		return nil, nil, err
	}
	// parse table
	news, blogs, err := parseNewsAndBlogs(page)
	if err != nil {
		slog.Error("failed to parse news and blogs", "err", err)
		return nil, nil, err
	}
	return news, blogs, nil
}
