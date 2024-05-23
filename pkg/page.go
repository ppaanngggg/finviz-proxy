package pkg

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/snwfdhmp/errlog"
	"io"
	"net/http"
)

func fetchFinvizPage(ctx context.Context, params string) ([]byte, error) {
	// request page
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, "https://finviz.com/screener.ashx?"+params, nil,
	)
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
	page, err := io.ReadAll(resp.Body)
	if errlog.Debug(err) {
		return nil, err
	}
	return page, nil
}
