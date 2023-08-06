package main

import (
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
		params, err := fetchParams(r.Context())
		if err != nil {
			logrus.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		render.JSON(w, r, params)
	})

	http.ListenAndServe(":8000", r)
}
