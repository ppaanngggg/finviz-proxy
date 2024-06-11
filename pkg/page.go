package pkg

import (
	"context"
	"github.com/pkg/errors"
	"io"
	"log/slog"
	"net/http"
)

func fetchFinvizPage(ctx context.Context, params string, isElite bool) ([]byte, error) {
	baseUrl := "https://finviz.com/screener.ashx?"
	if isElite {
		baseUrl = "https://elite.finviz.com/screener.ashx?"
	}
	// request page
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, baseUrl+params, nil,
	)
	if err != nil {
		slog.Error("fetchFinvizPage http new request", "err", err)
		return nil, err
	}
	req.Header.Set("User-Agent", "curl/7.88.1")
	client := newClient()
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("fetchFinvizPage http do", "err", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Error("fetchFinvizPage status code not ok", "code", resp.StatusCode)
		return nil, errors.New("fetchFinvizPage status code not ok")
	}
	page, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("fetchFinvizPage read all http resp body", "err", err)
		return nil, err
	}
	return page, nil
}
