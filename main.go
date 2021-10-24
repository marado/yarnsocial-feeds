package main

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	version bool
	debug   bool

	bind      string
	server    bool
	baseURL   string
	dataDir   string
	feedsFile string
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.BoolVarP(&version, "version", "v", false, "display version information")
	flag.BoolVarP(&debug, "debug", "D", false, "enable debug logging")

	flag.StringVarP(&bind, "bind", "b", "0.0.0.0:8000", "interface and port to bind to in server mode")
	flag.BoolVarP(&server, "server", "s", false, "enable server mode")
	flag.StringVarP(&dataDir, "data-dir", "d", "./data", "data directory to store feeds in")
	flag.StringVarP(&baseURL, "base-url", "u", "http://0.0.0.0:8000", "base url for generated urls")
	flag.StringVarP(&feedsFile, "feeds-file", "f", "feeds.yaml", "feeds configuration file in server mode")
}

func flagNameFromEnvironmentName(s string) string {
	s = strings.ToLower(s)
	s = strings.Replace(s, "_", "-", -1)
	return s
}

func parseArgs() error {
	for _, v := range os.Environ() {
		vals := strings.SplitN(v, "=", 2)
		flagName := flagNameFromEnvironmentName(vals[0])
		fn := flag.CommandLine.Lookup(flagName)
		if fn == nil || fn.Changed {
			continue
		}
		if err := fn.Value.Set(vals[1]); err != nil {
			return err
		}
	}
	flag.Parse()
	return nil
}

func main() {
	parseArgs()

	if version {
		fmt.Printf("feeds %s\n", FullVersion())
		os.Exit(0)
	}

	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if server {
		app, err := NewApp(
			WithBind(bind),
			WithDataDir(dataDir),
			WithBaseURL(baseURL),
			WithFeedsFile(feedsFile),
		)
		if err != nil {
			log.WithError(err).Fatal("error creating app for server mode")
		}
		if err := app.Run(); err != nil {
			log.WithError(err).Fatal("error running app")
		}
		os.Exit(0)
	}

	uri := flag.Arg(0)
	name := flag.Arg(1)

	conf := &Config{DataDir: "."}

	u, err := ParseURI(uri)
	if err != nil {
		log.WithError(err).Errorf("error parsing feed %s: %s", name, uri)
	} else {
		switch u.Type {
		case "rss", "http", "https":
			if err := UpdateRSSFeed(conf, name, uri); err != nil {
				log.WithError(err).Errorf("error updating rss feed %s: %s", name, uri)
			}
		case "twitter":
			if err := UpdateTwitterFeed(conf, name, u.Config); err != nil {
				log.WithError(err).Errorf("error updating twitter feed %s: %s", name, uri)
			}
		default:
			log.Warnf("error unknown feed type %s: %s", name, uri)
		}
	}
}
