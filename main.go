package main

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
	"github.com/snwfdhmp/errlog"
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

func parseKeyValuePairs(str string) map[string]string {
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

func parseFilterNameAndDescription(td *goquery.Selection) (name string, description string) {
	/*
		<td width="10%" align="center">
		    <span class="screener-combo-title"
		          style="cursor:pointer;"
		          data-boxover="cssbody=[tooltip_bdy] cssheader=[tooltip_hdr] header=[Exchange] body=[<table width=300><tr><td class='tooltip_tab'>Stock Exchange at which a stock is listed.</td></tr></table>] delay=[500]">
		        Exchange
		    </span>
		</td>
	*/
	span := td.Find("span").First()
	if span == nil || span.Length() == 0 {
		html, err := td.Html()
		logrus.WithField("td", html).WithField("err", err).Warning("span not found in td")
		return
	}
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
	td = doc.Find("td").First()
	if td == nil || td.Length() == 0 {
		logrus.WithField("body", body).Warning("td not found in body")
		return
	}
	description = td.Text()
	return
}

func parseFilterOptions(td *goquery.Selection) []FilterOption {
	selection := td.Find("select").First()
	if selection == nil || selection.Length() == 0 {
		html, err := td.Html()
		logrus.WithField("td", html).WithField("err", err).Warning("select not found in td")
		return nil
	}
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
		name := option.Text()
		value := option.AttrOr("value", "")
		if value == "" || value == "model" { // ignore Any and Custom (Elite only)
			options = append(options, FilterOption{
				Name: name, Value: prefix + "_" + value,
			})
		}
	})
	return options
}

func fetchFilters() ([]Filter, error) {
	// request page and get filters table
	req, err := http.NewRequest(http.MethodGet, "https://finviz.com/screener.ashx?ft=4", nil)
	if errlog.Debug(err) {
		return nil, err
	}
	req.Header.Set("User-Agent", "curl/7.88.1")
	resp, err := http.DefaultClient.Do(req)
	if errlog.Debug(err) {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logrus.Error("status code not ok")
		return nil, errors.New("status code not ok")
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if errlog.Debug(err) {
		return nil, err
	}
	tbody := doc.Find("#filter-table-filters > tbody").First()
	if tbody == nil || tbody.Length() == 0 {
		logrus.Error("tbody not found")
		return nil, errors.New("tbody not found")
	}
	// parse filters, each filter is a meta and an option
	metaTds := make([]*goquery.Selection, 0)
	optionsTds := make([]*goquery.Selection, 0)
	tbody.Find("tr").Each(func(i int, tr *goquery.Selection) {
		tr.Find("td").Each(func(j int, td *goquery.Selection) {
			switch j % 2 {
			case 0:
				metaTds = append(metaTds, td)
			case 1:
				optionsTds = append(optionsTds, td)
			}
		})
	})
	if len(metaTds) != len(optionsTds) {
		logrus.Error("len(metaTds) != len(optionsTds)")
		return nil, errors.New("len(metaTds) != len(optionsTds)")
	}
	filters := make([]Filter, 0)
	// parse meta and optionsTds to get filters
	for i := 0; i < len(metaTds); i++ {
		metaTd := metaTds[i]
		optionTd := optionsTds[i]
		name, description := parseFilterNameAndDescription(metaTd)
		if name != "" {
			options := parseFilterOptions(optionTd)
			if len(options) > 0 {
				logrus.WithField("Index", i).
					WithField("Name", name).
					WithField("Description", description).
					WithField("Options", options).
					Info("Filter Added")
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

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true, FullTimestamp: true})
	fetchFilters()
}
