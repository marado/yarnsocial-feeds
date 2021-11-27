package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

type App struct {
	conf *Config
	cron *cron.Cron
}

func NewApp(options ...Option) (*App, error) {
	conf := NewConfig()

	for _, opt := range options {
		if err := opt(conf); err != nil {
			return nil, err
		}
	}

	if Exists(conf.FeedsFile) {
		if err := conf.LoadFeeds(); err != nil {
			log.WithError(err).Error("error loading feeds")
			return nil, fmt.Errorf("error loading feeds: %w", err)
		}
	}

	cron := cron.New()

	return &App{conf: conf, cron: cron}, nil
}

func (app *App) initRoutes() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/", app.IndexHandler).Methods(http.MethodGet, http.MethodHead, http.MethodPost)
	router.HandleFunc("/health", app.HealthHandler).Methods(http.MethodGet, http.MethodHead)
	router.HandleFunc("/feeds", app.FeedsHandler).Methods(http.MethodGet, http.MethodHead)
	router.HandleFunc("/we-are-feeds.txt", app.WeAreFeedsHandler).Methods(http.MethodGet, http.MethodHead)
	router.HandleFunc("/{name}/twtxt.txt", app.FeedHandler).Methods(http.MethodGet, http.MethodHead)
	router.HandleFunc("/{name}/avatar.png", app.AvatarHandler).Methods(http.MethodGet, http.MethodHead)

	return router
}

func (app *App) setupCronJobs() error {
	for name, jobSpec := range Jobs {
		if jobSpec.Schedule == "" {
			continue
		}

		job := jobSpec.Factory(app.conf)
		if err := app.cron.AddJob(jobSpec.Schedule, job); err != nil {
			return err
		}
		log.Infof("Started background job %s (%s)", name, jobSpec.Schedule)
	}

	return nil
}

func (app *App) signalHandler(ch chan os.Signal) {
	for sig := range ch {
		switch sig {
		case syscall.SIGHUP:
			log.Info("reloading feeds on SIGHUP")
			if err := app.conf.LoadFeeds(); err != nil {
				log.WithError(err).Warn("error reloading feeds")
			}
		default:
			log.Warnf("ignoring unhandled signal %s", sig)
		}
	}
}

func (app *App) setupSignalHandlers() error {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGHUP)

	go app.signalHandler(ch)

	return nil
}

func (app *App) runStartupJobs() {
	time.Sleep(time.Second * 5)

	log.Info("running startup jobs")
	for name, jobSpec := range StartupJobs {
		job := jobSpec.Factory(app.conf)
		log.Infof("running %s now...", name)
		job.Run()
	}
}

func (app *App) GetFeeds() (feeds []Feed) {
	files, err := WalkMatch(app.conf.DataDir, "*.txt")
	if err != nil {
		log.WithError(err).Error("error reading feeds directory")
		return nil
	}

	for _, filename := range files {
		name := BaseWithoutExt(filename)

		stat, err := os.Stat(filename)
		if err != nil {
			log.WithError(err).Warnf("error getting feed stats for %s", name)
			continue
		}
		lastModified := humanize.Time(stat.ModTime())

		uri := fmt.Sprintf("%s/%s/twtxt.txt", app.conf.BaseURL, name)
		if feed, ok := app.conf.Feeds[name]; ok {
			feed.URI = uri
			feed.LastModified = lastModified
			feeds = append(feeds, *feed)
		}
	}

	sort.Slice(feeds, func(i, j int) bool { return feeds[i].Name < feeds[j].Name })

	return
}

func (app *App) Run() error {
	router := app.initRoutes()

	if err := app.setupCronJobs(); err != nil {
		log.WithError(err).Error("error setting up background jobs")
		return err
	}
	app.cron.Start()
	log.Info("started background jobs")

	if err := app.setupSignalHandlers(); err != nil {
		log.WithError(err).Error("error setting up signal handlers")
		return err
	}
	log.Info("setup signal handlers")

	log.Infof("feeds %s listening on http://%s", FullVersion(), app.conf.Addr)

	go app.runStartupJobs()

	return http.ListenAndServe(app.conf.Addr, router)
}
