package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"github.com/snwfdhmp/errlog"
	"net/http"
	"time"
)

type config struct {
	LogColor bool          `default:"true"`
	Timeout  time.Duration `default:"60s"`
	Throttle int           `default:"100"`
}

var globalParams *Params

func init() {
	var err error
	globalParams, err = fetchParams(context.Background())
	if errlog.Debug(err) {
		panic(err)
	}
	go func() {
		for {
			time.Sleep(time.Hour)
			params, err := fetchParams(context.Background())
			if errlog.Debug(err) {
				continue
			} else {
				globalParams = params
			}
		}
	}()
}

func main() {
	var c config
	if err := envconfig.Process("", &c); errlog.Debug(err) {
		return
	}

	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: c.LogColor, FullTimestamp: true})
	middleware.DefaultLogger = middleware.RequestLogger(
		&middleware.DefaultLogFormatter{Logger: logrus.StandardLogger(), NoColor: !c.LogColor},
	)

	r := chi.NewRouter()
	r.Use(middleware.Timeout(c.Timeout))
	r.Use(middleware.Throttle(c.Throttle))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/params", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, globalParams)
	})

	logrus.Infof("Listening on :8000")
	http.ListenAndServe(":8000", r)
}
