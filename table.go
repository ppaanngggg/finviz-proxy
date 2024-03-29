package main

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"github.com/snwfdhmp/errlog"
)

type TableParams struct {
	Order   string   `json:"order"`
	Desc    bool     `json:"desc"`
	Signal  string   `json:"signal"`
	Filters []string `json:"filters"`
}

func NewTableParams(query map[string][]string) *TableParams {
	params := &TableParams{}
	if order, ok := query["order"]; ok {
		if len(order) > 0 {
			params.Order = order[0]
		}
	}
	if desc, ok := query["desc"]; ok {
		if len(desc) > 0 && (desc[0] == "1" || strings.ToLower(desc[0]) == "true") {
			params.Desc = true
		}
	}
	if signal, ok := query["signal"]; ok {
		if len(signal) > 0 {
			params.Signal = signal[0]
		}
	}
	for k, v := range query {
		if k == "filters" || strings.HasPrefix(k, "filters[") {
			params.Filters = append(params.Filters, v...)
		}
	}
	return params
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

func parseTable(page []byte) (*Table, error) {
	// parse table from page
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if errlog.Debug(err) {
		return nil, err
	}
	// build table
	table := &Table{}
	thead := doc.Find("#screener-table").Find("thead")
	thead.Find("th").Each(
		func(i int, th *goquery.Selection) {
			val, exists := th.Attr("class")
			if exists && strings.Contains(val, "header") {
				table.Headers = append(table.Headers, strings.TrimSpace(th.Text()))
			}
		},
	)
	tbody := thead.SiblingsFiltered("tbody")
	tbody.Find("tr").Each(
		func(i int, tr *goquery.Selection) {
			buf := make([]string, 0, len(table.Headers))
			tr.Find("td").Each(
				func(i int, td *goquery.Selection) {
					buf = append(buf, strings.TrimSpace(td.Text()))
				},
			)
			table.Rows = append(table.Rows, buf)
		},
	)
	return table, nil
}

func fetchPageAndParseTable(ctx context.Context, params *TableParams) (*Table, error) {
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
	// parse table
	table, err := parseTable(page)
	if errlog.Debug(err) {
		return nil, err
	}
	// cache table
	tableCache.Set(uri, table, cache.DefaultExpiration)
	return table, nil
}
