package main

import (
	"bytes"
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"github.com/snwfdhmp/errlog"
	"strings"
	"time"
)

type TableParams struct {
	Order   string   `json:"order"`
	Desc    bool     `json:"desc"`
	Signal  string   `json:"signal"`
	Filters []string `json:"filters"`
}

type Table struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

func buildUri(params *TableParams) string {
	ret := ""
	if params.Order != "" {
		if params.Desc {
			ret += "o=-" + params.Order
		} else {
			ret += "o=" + params.Order
		}
	}
	if params.Signal != "" {
		if ret != "" {
			ret += "&"
		}
		ret += "s=" + params.Signal
	}
	if len(params.Filters) > 0 {
		if ret != "" {
			ret += "&"
		}
		ret += "f="
		for i, filter := range params.Filters {
			if i != 0 {
				ret += ","
			}
			ret += filter
		}
	}
	logrus.Infof("buildUri: %s", ret)
	return ret
}

var tableCache *cache.Cache

func init() {
	tableCache = cache.New(time.Minute, time.Minute)
}

func parseTable(ctx context.Context, params *TableParams) (*Table, error) {
	uri := buildUri(params)
	// check cache
	if table, found := tableCache.Get(uri); found {
		return table.(*Table), nil
	}
	// fetch page
	page, err := fetchFinvizPage(ctx, uri)
	if errlog.Debug(err) {
		return nil, err
	}
	// parse table from page
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if errlog.Debug(err) {
		return nil, err
	}

	table := &Table{}
	node := doc.Find("#screener-table .table-light")
	node.Find("tr").Each(func(i int, tr *goquery.Selection) {
		buf := make([]string, 0)
		tr.Find("td").Each(func(j int, td *goquery.Selection) {
			buf = append(buf, strings.TrimSpace(td.Text()))
		})
		if i == 0 {
			table.Headers = buf
		} else {
			table.Rows = append(table.Rows, buf)
		}
	})

	// cache table
	tableCache.Set(uri, table, cache.DefaultExpiration)
	return table, nil
}
