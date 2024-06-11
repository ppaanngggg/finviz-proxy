package pkg

import (
	"bytes"
	"context"
	"encoding/csv"
	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"io"
	"log/slog"
	"strings"
)

type TableParams struct {
	Order   string   `json:"order"`
	Desc    bool     `json:"desc"`
	Signal  string   `json:"signal"`
	Filters []string `json:"filters"`
}

func (p *TableParams) BuildUri() string {
	ret := ""
	if p.Order != "" {
		if p.Desc {
			ret += "o=-" + p.Order
		} else {
			ret += "o=" + p.Order
		}
	}
	if p.Signal != "" {
		if ret != "" {
			ret += "&"
		}
		ret += "s=" + p.Signal
	}
	if len(p.Filters) > 0 {
		if ret != "" {
			ret += "&"
		}
		ret += "f="
		for i, filter := range p.Filters {
			if i != 0 {
				ret += ","
			}
			ret += filter
		}
	}
	return ret
}

func checkSorter(allowParams *Params, order string) bool {
	for _, sorter := range allowParams.Sorters {
		if sorter.Value == order {
			return true
		}
	}
	return false
}

func checkSignal(allowParams *Params, signal string) bool {
	for _, s := range allowParams.Signals {
		if s.Value == signal {
			return true
		}
	}
	return false
}

func checkFilter(allowParams *Params, filter string) bool {
	for _, f := range allowParams.Filters {
		for _, o := range f.Options {
			if o.Value == filter {
				return true
			}
		}
	}
	return false
}

func ParseTableParams(allowParams *Params, query map[string][]string) (*TableParams, error) {
	for k := range query {
		if k != "order" && k != "desc" && k != "signal" && k != "auth" &&
			k != "filters" && !strings.HasPrefix(k, "filters[") {
			return nil, errors.Errorf("invalid query key: %s", k)
		}
	}

	params := &TableParams{}
	if order, ok := query["order"]; ok {
		if len(order) > 0 {
			if !checkSorter(allowParams, order[0]) {
				return nil, errors.Errorf("invalid order: %s", order[0])
			}
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
			if !checkSignal(allowParams, signal[0]) {
				return nil, errors.Errorf("invalid signal: %s", signal[0])
			}
			params.Signal = signal[0]
		}
	}
	for k, v := range query {
		if k == "filters" || strings.HasPrefix(k, "filters[") {
			for _, filter := range v {
				if !checkFilter(allowParams, filter) {
					return nil, errors.Errorf("invalid filter: %s", filter)
				}
			}
			params.Filters = append(params.Filters, v...)
		}
	}
	return params, nil
}

type Table struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

func parseTable(page []byte) (*Table, error) {
	// parse table from page
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if err != nil {
		slog.Error("failed to parse table from page", "err", err)
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

func FetchPageAndParseTable(ctx context.Context, uri string, isElite bool) (*Table, error) {
	// fetch page
	page, err := fetchFinvizPage(ctx, uri, isElite)
	if err != nil {
		slog.Error("failed to fetch page", "err", err)
		return nil, err
	}
	// parse table
	table, err := parseTable(page)
	if err != nil {
		slog.Error("failed to parse table", "err", err)
		return nil, err
	}
	return table, nil
}

type TableParamsWithAPIKey struct {
	TableParams
	ApiKey string `json:"apiKey"`
}

func ParseTableParamsWithAPIKey(allowParams *Params, query map[string][]string) (*TableParamsWithAPIKey, error) {
	params, err := ParseTableParams(allowParams, query)
	if err != nil {
		return nil, err
	}
	// find apiKey from query
	apiKeys, ok := query["auth"]
	if !ok {
		return nil, errors.New("apiKeys not found")
	}
	if len(apiKeys) == 0 {
		return nil, errors.New("apiKeys is empty")
	}
	apiKey := apiKeys[0]
	if apiKey == "" {
		return nil, errors.New("apiKey is empty")
	}
	return &TableParamsWithAPIKey{
		TableParams: *params,
		ApiKey:      apiKey,
	}, nil
}

func (p *TableParamsWithAPIKey) BuildUri() string {
	ret := p.TableParams.BuildUri()
	if ret != "" {
		ret += "&"
	}
	ret += "auth=" + p.ApiKey
	return ret
}

func ExportTable(ctx context.Context, uri string) (*Table, error) {
	// fetch csv
	csvBytes, err := fetchExportCSV(ctx, uri)
	if err != nil {
		slog.Error("failed to fetch csv", "err", err)
		return nil, err
	}
	// parse table
	r := csv.NewReader(bytes.NewReader(csvBytes))
	table := &Table{}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error("failed to read csv", "err", err)
		}
		if len(table.Headers) == 0 {
			table.Headers = record
		} else {
			table.Rows = append(table.Rows, record)
		}
	}
	return table, nil
}
