package pkg

import (
	"context"
	"github.com/pkg/errors"
	"io"
	"log/slog"
	"net/http"
	"time"
)

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
	client := &http.Client{
		Timeout: time.Minute,
	}
	resp, err := client.Do(req)
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
