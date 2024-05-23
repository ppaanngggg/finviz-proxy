package main

import (
	"context"
	"github.com/ppaanngggg/finviz-proxy/pkg"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"github.com/snwfdhmp/errlog"
)

type config struct {
	LogColor bool          `default:"true"`
	Port     int           `default:"8000"`
	Timeout  time.Duration `default:"60s"`
	Throttle int           `default:"100"`
}

var (
	c            config
	globalParams *pkg.Params
)

func init() {
	// load config
	if err := envconfig.Process("", &c); errlog.Debug(err) {
		panic(err)
	}
	// setup logger
	logrus.SetFormatter(
		&logrus.TextFormatter{
			ForceColors: c.LogColor, FullTimestamp: true,
		},
	)
	middleware.DefaultLogger = middleware.RequestLogger(
		&middleware.DefaultLogFormatter{
			Logger: logrus.StandardLogger(), NoColor: !c.LogColor,
		},
	)
	// fetch params
	func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
		defer cancel()
		params, err := pkg.FetchParams(ctx)
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
				params, err := pkg.FetchParams(ctx)
				if errlog.Debug(err) {
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
		"/table", func(w http.ResponseWriter, r *http.Request) {
			params, err := pkg.ParseTableParams(globalParams, r.URL.Query())
			if errlog.Debug(err) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
			table, err := pkg.FetchPageAndParseTable(r.Context(), params)
			if errlog.Debug(err) {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			render.JSON(w, r, table)
		},
	)

	addr := ":" + strconv.Itoa(c.Port)
	logrus.Infof("Listening on " + addr)
	http.ListenAndServe(addr, r)
}
