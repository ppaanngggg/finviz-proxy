package pkg

import (
	"context"
	"github.com/pkg/errors"
	"io"
	"log/slog"
	"net/http"
)

func fetchFinvizPage(ctx context.Context, params string) ([]byte, error) {
	// request page
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, "https://finviz.com/screener.ashx?"+params, nil,
	)
	if err != nil {
		slog.Error("fetchFinvizPage http new request", "err", err)
		return nil, err
	}
	req.Header.Set("User-Agent", "curl/7.88.1")
	resp, err := http.DefaultClient.Do(req)
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

func fetchExportCSV(ctx context.Context, params string) ([]byte, error) {
	// request csv
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, "https://elite.finviz.com/export.ashx?"+params, nil,
	)
	if err != nil {
		slog.Error("fetchExportCSV http new request", "err", err)
		return nil, err
	}
	req.Header.Set("User-Agent", "curl/7.88.1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("fetchExportCSV http do", "err", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Error("fetchExportCSV status code not ok", "code", resp.StatusCode)
		return nil, errors.New("fetchExportCSV status code not ok")
	}
	csv, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("fetchExportCSV read all http resp body", "err", err)
		return nil, err
	}
	return csv, nil
}
