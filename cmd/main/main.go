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
	Port       int           `default:"8000"`
	Timeout    time.Duration `default:"60s"`
	Throttle   int           `default:"100"`
	CacheTTL   time.Duration `default:"60s"`
	EliteLogin bool          `default:"false"`
	Email      string        `default:""`
	Password   string        `default:""`
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
	// elite login
	if c.EliteLogin {
		if c.Email == "" || c.Password == "" {
			panic("email or password is empty")
		}
		ok, err := pkg.EliteLogin(context.Background(), c.Email, c.Password)
		if err != nil {
			panic(err)
		}
		if !ok {
			panic("login failed")
		}
		slog.Info("login success")
		go func() {
			for {
				time.Sleep(24 * time.Hour)
				func() {
					slog.Info("login...")
					ok, err = pkg.EliteLogin(context.Background(), c.Email, c.Password)
					if err != nil {
						slog.Error("login err", "err", err)
						return
					}
					if !ok {
						slog.Error("login failed")
						return
					}
					slog.Info("login success")
				}()
			}
		}()
	}
	// fetch params
	func() {
		params, err := pkg.FetchParams(context.Background(), c.EliteLogin)
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
				params, err := pkg.FetchParams(context.Background(), c.EliteLogin)
				if err != nil {
					slog.Error("fetch params err", "err", err)
					return
				}
				globalParams = params
				slog.Info("fetch params success")
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
				render.Status(r, http.StatusBadRequest)
				if pkg.IsParamsError(err) {
					render.JSON(w, r, err)
				} else {
					render.PlainText(w, r, err.Error())
				}
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
				render.Status(r, http.StatusBadRequest)
				if pkg.IsParamsError(err) {
					render.JSON(w, r, err)
				} else {
					render.PlainText(w, r, err.Error())
				}
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
			table, err := pkg.FetchPageAndParseTable(r.Context(), uri, c.EliteLogin)
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
	r.Post(
		"/table_v2", func(w http.ResponseWriter, r *http.Request) {
			params, err := pkg.ParseTableParamsV2(globalParams, r.Body)
			defer r.Body.Close()
			if err != nil {
				slog.Error("parse table params v2", "err", err)
				render.Status(r, http.StatusBadRequest)
				if pkg.IsParamsError(err) {
					render.JSON(w, r, err)
				} else {
					render.PlainText(w, r, err.Error())
				}
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
			table, err := pkg.FetchPageAndParseTable(r.Context(), uri, c.EliteLogin)
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
				render.Status(r, http.StatusBadRequest)
				if pkg.IsParamsError(err) {
					render.JSON(w, r, err)
				} else {
					render.PlainText(w, r, err.Error())
				}
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
