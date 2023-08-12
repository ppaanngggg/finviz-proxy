package main

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"github.com/snwfdhmp/errlog"
	"io"
	"net/http"
	"os"
	"time"
)

type config struct {
	LogColor bool          `default:"true"`
	Timeout  time.Duration `default:"60s"`
	Throttle int           `default:"100"`
}

var (
	c            config
	globalParams *Params
)

func init() {
	// load config
	if err := envconfig.Process("", &c); errlog.Debug(err) {
		panic(err)
	}
	// setup logger
	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: c.LogColor, FullTimestamp: true})
	middleware.DefaultLogger = middleware.RequestLogger(
		&middleware.DefaultLogFormatter{Logger: logrus.StandardLogger(), NoColor: !c.LogColor},
	)
	// fetch params
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
		defer cancel()
		params, err := fetchParams(ctx)
		if errlog.Debug(err) {
			panic(err)
		}
		globalParams = params
	}()
	go func() {
		for {
			time.Sleep(time.Hour)
			func() {
				logrus.Info("Fetching params...")
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
				defer cancel()
				params, err := fetchParams(ctx)
				if errlog.Debug(err) {
					return
				}
				globalParams = params
			}()
		}
	}()
}

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

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Timeout(c.Timeout))
	r.Use(middleware.Throttle(c.Throttle))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/download", func(w http.ResponseWriter, r *http.Request) {
		html, err := fetchFinvizPage(r.Context(), "")
		if errlog.Debug(err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		os.WriteFile("screener.ashx.html", html, 0644)
	})
	r.Get("/params", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, globalParams)
	})
	r.Post("/table", func(w http.ResponseWriter, r *http.Request) {
		params := &TableParams{}
		if err := render.DecodeJSON(r.Body, params); errlog.Debug(err) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		table, err := parseTable(r.Context(), params)
		if errlog.Debug(err) {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		render.JSON(w, r, table)
	})

	logrus.Infof("Listening on :8000")
	http.ListenAndServe(":8000", r)
}
