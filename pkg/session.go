package pkg

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

var cookies, _ = cookiejar.New(nil)

func newClient() *http.Client {
	return &http.Client{
		Jar:     cookies,
		Timeout: time.Minute,
	}
}

func EliteLogin(ctx context.Context, email string, password string) (bool, error) {
	// build request
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, "https://finviz.com/login_submit.ashx",
		bytes.NewReader([]byte(url.PathEscape(fmt.Sprintf("email=%s&password=%s", email, password)))),
	)
	if err != nil {
		slog.Error("login http new request failed", "err", err)
		return false, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// send request
	client := newClient()
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("login http do failed", "err", err)
		return false, err
	}
	defer resp.Body.Close()
	// check response
	if resp.StatusCode != http.StatusOK {
		slog.Error("login http status != 200", "status", resp.StatusCode)
		return false, fmt.Errorf("login http status: %d", resp.StatusCode)
	}
	// check new request URL
	req = resp.Request
	if req == nil {
		slog.Error("login http redirect request is nil")
		return false, fmt.Errorf("login http redirect request is nil")
	}
	if req.URL.Host != "elite.finviz.com" {
		slog.Error("login http redirect url host != elite.finviz.com", "host", req.URL.Host)
		return false, fmt.Errorf("login http redirect url host: %s", req.URL.Host)
	}
	return true, nil
}
