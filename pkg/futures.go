package pkg

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"log/slog"
	"net/http"
)

type FutureQuota struct {
	Label     string  `json:"label"`
	Ticker    string  `json:"ticker"`
	Last      float64 `json:"last"`
	Change    float64 `json:"change"`
	PrevClose float64 `json:"prevClose"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
}

func FetchAllFutures(ctx context.Context, isElite bool) (map[string]FutureQuota, error) {
	url := "https://finviz.com/api/futures_all.ashx?timeframe=NO"
	if isElite {
		url = "https://elite.finviz.com/api/futures_all.ashx?timeframe=NO"
	}
	// request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Error("FetchAllFutures http new request", "err", err)
		return nil, err
	}
	req.Header.Set("User-Agent", "curl/7.88.1")
	client := newClient()
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("FetchAllFutures http do", "err", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Error("FetchAllFutures status code not ok", "status", resp.StatusCode)
		return nil, errors.New("FetchAllFutures status code not ok")
	}
	// unmarshal to map
	ret := make(map[string]FutureQuota)
	err = json.NewDecoder(resp.Body).Decode(&ret)
	if err != nil {
		slog.Error("FetchAllFutures json decode response", "err", err)
		return nil, err
	}
	return ret, nil
}
