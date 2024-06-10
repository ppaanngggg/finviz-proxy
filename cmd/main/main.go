package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/patrickmn/go-cache"
	"github.com/ppaanngggg/finviz-proxy/pkg"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/kelseyhightower/envconfig"
)

type config struct {
	Port     int           `default:"8000"`
	Timeout  time.Duration `default:"60s"`
	Throttle int           `default:"100"`
	CacheTTL time.Duration `default:"60s"`
}

var (
	c            config
	globalParams *pkg.Params
	tableCache   *cache.Cache
)

func init() {
	// load config
	if err := envconfig.Process("", &c); err != nil {
		panic(err)
	}
	// init cache
	tableCache = cache.New(c.CacheTTL, c.CacheTTL)
	// fetch params
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
		defer cancel()
		params, err := pkg.FetchParams(ctx)
		if err != nil {
			panic(err)
		}
		globalParams = params
	}()
	go func() {
		for {
			time.Sleep(time.Hour)
			func() {
				slog.Info("fetching params...")
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
				defer cancel()
				params, err := pkg.FetchParams(ctx)
				if err != nil {
					slog.Error("fetch params", "err", err)
					return
				}
				globalParams = params
			}()
		}
	}()
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Timeout(c.Timeout))
	r.Use(middleware.Throttle(c.Throttle))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get(
		"/params", func(w http.ResponseWriter, r *http.Request) {
			render.JSON(w, r, globalParams)
		},
	)
	r.Get(
		"/filter", func(w http.ResponseWriter, r *http.Request) {
			params, err := pkg.ParseTableParams(globalParams, r.URL.Query())
			if err != nil {
				slog.Error("parse table params", "err", err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
			uri := params.BuildUri()
			render.PlainText(w, r, uri)
		},
	)
	r.Get(
		"/table", func(w http.ResponseWriter, r *http.Request) {
			params, err := pkg.ParseTableParams(globalParams, r.URL.Query())
			if err != nil {
				slog.Error("parse table params", "err", err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
			uri := params.BuildUri()
			slog.Info("to fetch page", "uri", uri)
			// check cache
			if table, found := tableCache.Get(uri); found {
				render.JSON(w, r, table)
				return
			}
			// fetch page and parse table
			table, err := pkg.FetchPageAndParseTable(r.Context(), uri)
			if err != nil {
				slog.Error("fetch page and parse table", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			// cache table
			tableCache.Set(uri, table, cache.DefaultExpiration)
			render.JSON(w, r, table)
		},
	)
	r.Get(
		"/elite_table", func(w http.ResponseWriter, r *http.Request) {
			params, err := pkg.ParseTableParamsWithAPIKey(globalParams, r.URL.Query())
			if err != nil {
				slog.Error("parse elite table params", "err", err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
			uri := params.BuildUri()
			slog.Info("to fetch elite table", "uri", uri)
			// check cache
			if table, found := tableCache.Get(uri); found {
				render.JSON(w, r, table)
				return
			}
			// fetch csv
			table, err := pkg.ExportTable(r.Context(), uri)
			if err != nil {
				slog.Error("export table", "err", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			// cache table
			tableCache.Set(uri, table, cache.DefaultExpiration)
			render.JSON(w, r, table)
		},
	)

	addr := ":" + strconv.Itoa(c.Port)
	slog.Info("Listening on", "addr", addr)
	http.ListenAndServe(addr, r)
}
