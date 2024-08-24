package main

import (
	"context"
	"encoding/json"
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
	c             config
	globalParams  *pkg.Params
	globalFutures map[string]pkg.FutureQuota
	globalNews    []pkg.Record
	globalBlogs   []pkg.Record
	tableCache    *cache.Cache
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
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
				defer cancel()
				params, err := pkg.FetchParams(ctx, c.EliteLogin)
				if err != nil {
					slog.Error("fetch params err", "err", err)
					return
				}
				globalParams = params
				slog.Info("fetch params success")
			}()
		}
	}()
	// fetch futures
	func() {
		futures, err := pkg.FetchAllFutures(context.Background(), c.EliteLogin)
		if err != nil {
			panic(err)
		}
		globalFutures = futures
	}()
	go func() {
		for {
			time.Sleep(10 * time.Second)
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
				defer cancel()
				futures, err := pkg.FetchAllFutures(ctx, c.EliteLogin)
				if err != nil {
					slog.Error("fetch all futures err", "err", err)
					return
				}
				globalFutures = futures
				slog.Info("fetch all futures success")
			}()
		}
	}()
	// fetch news and blogs
	func() {
		news, blogs, err := pkg.FetchAndParseNewsAndBlogs(context.Background(), c.EliteLogin)
		if err != nil {
			panic(err)
		}
		globalNews = news
		globalBlogs = blogs
	}()
	go func() {
		for {
			time.Sleep(time.Minute)
			func() {
				ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
				defer cancel()
				news, blogs, err := pkg.FetchAndParseNewsAndBlogs(ctx, c.EliteLogin)
				if err != nil {
					slog.Error("fetch all news and blogs err", "err", err)
					return
				}
				globalNews = news
				globalBlogs = blogs
				slog.Info("fetch all news and blogs success")
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

	/*
		stock screener apis
	*/

	r.Get(
		"/params", func(w http.ResponseWriter, r *http.Request) {
			render.JSON(w, r, globalParams)
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

	/*
		futures apis
	*/

	r.Get("/futures/all", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, globalFutures)
	})

	r.Post("/futures", func(w http.ResponseWriter, r *http.Request) {
		symbols := struct {
			Symbols []string `json:"symbols"`
		}{}
		if err := json.NewDecoder(r.Body).Decode(&symbols); err != nil {
			slog.Error("parse futures json", "err", err)
			render.Status(r, http.StatusBadRequest)
			render.PlainText(w, r, `request body should be as: {"symbols": [...]}`)
			return
		}
		defer r.Body.Close()
		// build ret
		ret := struct {
			Futures []pkg.FutureQuota `json:"futures"`
		}{}
		for _, symbol := range symbols.Symbols {
			flag := false
			for _, v := range globalFutures {
				if v.Label == symbol {
					ret.Futures = append(ret.Futures, v)
					flag = true
					break
				}
			}
			if !flag {
				slog.Error("can't find symbol in all futures", "symbol", symbol)
				render.Status(r, http.StatusBadRequest)
				render.PlainText(w, r, "can't find symbol: "+symbol)
				return
			}
		}
		render.JSON(w, r, ret)
	})

	/*
		news and blogs api
	*/

	r.Get("/news", func(w http.ResponseWriter, r *http.Request) {
		ret := struct {
			News []pkg.Record `json:"news"`
		}{}
		ret.News = globalNews
		render.JSON(w, r, ret)
	})

	r.Get("/blogs", func(w http.ResponseWriter, r *http.Request) {
		ret := struct {
			Blogs []pkg.Record `json:"blogs"`
		}{}
		ret.Blogs = globalBlogs
		render.JSON(w, r, ret)
	})

	// start serve
	addr := ":" + strconv.Itoa(c.Port)
	slog.Info("Listening on", "addr", addr)
	http.ListenAndServe(addr, r)
}
