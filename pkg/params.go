package pkg

import (
	"bytes"
	"context"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
	"github.com/snwfdhmp/errlog"
	"regexp"
	"strings"
)

type FilterOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Filter struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Options     []FilterOption `json:"options"`
}

type Sorter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Signal struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Params struct {
	Filters []Filter `json:"filters"`
	Sorters []Sorter `json:"sorters"`
	Signals []Signal `json:"signals"`
}

func parseKeyValuePairs(str string) map[string]string {
	// cssbody=[tooltip_bdy] cssheader=[tooltip_hdr] header=[Exchange] body=[<table width=300><tr><td class='tooltip_tab'>Stock Exchange at which a stock is listed.</td></tr></table>] delay=[500]
	keyValuePairs := make(map[string]string)

	regex := regexp.MustCompile(`(\w+)=\[(.*?)\]`)
	matches := regex.FindAllStringSubmatch(str, -1)

	for _, match := range matches {
		key := match[1]
		value := strings.Trim(match[2], "[]")
		keyValuePairs[key] = value
	}

	return keyValuePairs
}

func parseUrlParams(str string) map[string]string {
	// screener.ashx?v=111&ft=4&o=tickersfilter
	urlParams := make(map[string]string)
	if strings.Contains(str, "?") {
		str = strings.Split(str, "?")[1]
		for _, param := range strings.Split(str, "&") {
			pair := strings.Split(param, "=")
			if len(pair) == 2 {
				urlParams[pair[0]] = pair[1]
			}
		}
	}
	return urlParams
}

func parseFilterNameAndDescription(span *goquery.Selection) (name string, description string) {
	/*
	   <span class="screener-combo-title"
	         style="cursor:pointer;"
	         data-boxover="cssbody=[tooltip_bdy] cssheader=[tooltip_hdr] header=[Exchange] body=[<table width=300><tr><td class='tooltip_tab'>Stock Exchange at which a stock is listed.</td></tr></table>] delay=[500]">
	       Exchange
	   </span>
	*/
	dataBoxover, exist := span.Attr("data-boxover")
	if !exist {
		html, err := span.Html()
		logrus.WithField("span", html).WithField("err", err).Warning("data-boxover not found in span")
		return
	}
	m := parseKeyValuePairs(dataBoxover)
	// set header as name
	name = m["header"]
	if name == "" {
		// if header not found, use span text
		name = span.Text()
	}
	// parse body to get description
	body := m["body"]
	if body == "" {
		logrus.WithField("data-boxover", dataBoxover).Warning("body not found in data-boxover")
		return
	}
	/*
		<table width=300><tr><td class='tooltip_tab'>Stock Exchange at which a stock is listed.</td></tr></table>
	*/
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if errlog.Debug(err) {
		return
	}
	td := doc.Find("td").First()
	if td == nil || td.Length() == 0 {
		logrus.WithField("body", body).Warning("td not found in body")
		return
	}
	description = td.Text()
	return
}

func parseFilterOptions(selection *goquery.Selection) []FilterOption {
	/*
		<select id="fs_exch" style="width: 100%; visibility: visible;"
		        class="screener-combo-text" onchange="ScreenerSelectOnChange(this)"
		        data-filter="exch" data-url="v=111&amp;ft=4"
		        data-url-selected="v=111&amp;f=exch_selected_filter&amp;ft=4"
		        data-selected="">
		    <option selected="selected" value="">Any</option>
		    <option value="amex">AMEX</option>
		    <option value="nasd">NASDAQ</option>
		    <option value="nyse">NYSE</option>
		    <option value="modal">Custom (Elite only)</option>
		</select>
	*/
	// extract data-filter as prefix
	prefix, exist := selection.Attr("data-filter")
	if !exist {
		html, err := selection.Html()
		logrus.WithField("selection", html).WithField("err", err).Warning("data-filter not found in selection")
		return nil
	}
	// iter options of select
	options := make([]FilterOption, 0)
	selection.Find("option").Each(func(i int, option *goquery.Selection) {
		// <option value="amex">AMEX</option>
		name := option.Text()
		if name == "Any" || name == "Custom (Elite only)" { // ignore Any and Custom (Elite only)
			return
		}
		value := option.AttrOr("value", "")
		options = append(options, FilterOption{
			Name: name, Value: prefix + "_" + value,
		})
	})
	return options
}

func parseFilters(doc *goquery.Document) ([]Filter, error) {
	table := doc.Find("table#filter-table-filters").First()
	if table == nil || table.Length() == 0 {
		logrus.Error("table not found")
		return nil, errors.New("table not found")
	}
	// parse filters, each filter is a meta and an option
	spans := make([]*goquery.Selection, 0)
	selections := make([]*goquery.Selection, 0)
	table.Find("span.screener-combo-title").Each(func(i int, span *goquery.Selection) {
		spans = append(spans, span)
	})
	table.Find("select.screener-combo-text").Each(func(i int, selection *goquery.Selection) {
		selections = append(selections, selection)
	})
	if len(spans) != len(selections) {
		logrus.Error("len(spans) != len(selections)")
		return nil, errors.New("len(spans) != len(selections)")
	}
	filters := make([]Filter, 0)
	// parse meta and selections to get filters
	for i := 0; i < len(spans); i++ {
		span := spans[i]
		selection := selections[i]
		name, description := parseFilterNameAndDescription(span)
		if name != "" {
			options := parseFilterOptions(selection)
			if len(options) > 0 {
				logrus.WithField("Index", i).
					WithField("Name", name).
					WithField("Description", description).
					WithField("Options", options).
					Debug("Filter Added")
				filters = append(filters, Filter{
					Name:        name,
					Description: description,
					Options:     options,
				})
			}
		}
	}
	return filters, nil
}

func parseSorters(doc *goquery.Document) ([]Sorter, error) {
	sorters := make([]Sorter, 0)
	doc.Find("select#orderSelect option").Each(func(i int, option *goquery.Selection) {
		name := option.Text()
		value, exists := option.Attr("value")
		if !exists {
			return
		}
		params := parseUrlParams(value)
		value = params["o"]
		if value != "" {
			sorters = append(sorters, Sorter{
				Name:  name,
				Value: value,
			})
		}
	})
	return sorters, nil
}

func parseSignals(doc *goquery.Document) ([]Signal, error) {
	signals := make([]Signal, 0)
	doc.Find("select#signalSelect option").Each(func(i int, option *goquery.Selection) {
		name := option.Text()
		if name == "None (all stocks)" {
			return
		}
		value, exists := option.Attr("value")
		if !exists {
			return
		}
		params := parseUrlParams(value)
		value = params["s"]
		if value != "" {
			signals = append(signals, Signal{
				Name:  name,
				Value: value,
			})
		}
	})
	return signals, nil
}

func FetchParams(ctx context.Context) (*Params, error) {
	page, err := fetchFinvizPage(ctx, "ft=4")
	if errlog.Debug(err) {
		return nil, err
	}

	// parse params from page
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if errlog.Debug(err) {
		return nil, err
	}
	params := &Params{}
	params.Filters, err = parseFilters(doc)
	if errlog.Debug(err) {
		return nil, err
	}
	params.Sorters, err = parseSorters(doc)
	if errlog.Debug(err) {
		return nil, err
	}
	params.Signals, err = parseSignals(doc)
	if errlog.Debug(err) {
		return nil, err
	}
	return params, nil
}
