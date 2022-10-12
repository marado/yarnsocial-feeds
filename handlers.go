package main

import (
	"bytes"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/aofei/cameron"
	"github.com/badgerodon/ioutil"
	"github.com/gorilla/mux"
	"github.com/rickb777/accept"
	log "github.com/sirupsen/logrus"
)

func render(name, tmpl string, ctx interface{}, w io.Writer) error {
	t, err := template.New(name).Parse(tmpl)
	if err != nil {
		return err
	}

	return t.Execute(w, ctx)
}

func renderMessage(w http.ResponseWriter, status int, title, message string) error {
	ctx := struct {
		Title   string
		Message string
	}{
		Title:   title,
		Message: message,
	}

	if err := render("message", messageTemplate, ctx, w); err != nil {
		return err
	}

	return nil
}

func (app *App) HealthHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	http.Error(w, "Healthy", http.StatusOK)
}

func (app *App) IndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead || r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html")

		ctx := struct {
			Title string
		}{
			Title: "RSS/Atom to twtxt feed aggregator service",
		}

		if r.Method == http.MethodHead {
			return
		}

		if err := render("index", indexTemplate, ctx, w); err != nil {
			log.WithError(err).Error("error rending index template")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if r.Method == http.MethodPost {
		uri := r.FormValue("uri")

		if uri == "" {
			if err := renderMessage(w, http.StatusBadRequest, "Error", "No uri supplied"); err != nil {
				log.WithError(err).Error("error rendering message template")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		u, err := ParseURI(uri)
		if err != nil {
			if err := renderMessage(w, http.StatusBadRequest, "Error", "Invalid URI"); err != nil {
				log.WithError(err).Error("error rendering message template")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		var feed Feed

		switch u.Type {
		case "rss", "http", "https":
			feed, err = ValidateRSSFeed(app.conf, uri)
		default:
			if err := renderMessage(w, http.StatusBadRequest, "Error", "Unsupproted feed"); err != nil {
				log.WithError(err).Error("error rendering message template")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		if err != nil {
			if err := renderMessage(w, http.StatusBadRequest, "Error", fmt.Sprintf("Unable to find a valid RSS/Atom feed for: %s", uri)); err != nil {
				log.WithError(err).Error("error rendering message template")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		if _, ok := app.conf.Feeds[feed.Name]; ok {
			if err := renderMessage(w, http.StatusConflict, "Error", "Feed already exists"); err != nil {
				log.WithError(err).Error("error rendering message template")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		app.conf.Feeds[feed.Name] = &feed
		if err := app.conf.SaveFeeds(); err != nil {
			msg := fmt.Sprintf("Could not save feed: %s", err)
			if err := renderMessage(w, http.StatusInternalServerError, "Error", msg); err != nil {
				log.WithError(err).Error("error rendering message template")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		msg := fmt.Sprintf("Feed successfully added %s: %s", feed.Name, feed.URI)
		if err := renderMessage(w, http.StatusCreated, "Success", msg); err != nil {
			log.WithError(err).Error("error rendering message template")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func (app *App) FeedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead || r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/plain")

		vars := mux.Vars(r)

		name := vars["name"]
		if name == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		fn := filepath.Join(app.conf.DataDir, fmt.Sprintf("%s.txt", name))
		if !Exists(fn) {
			log.Warnf("feed does not exist %s", name)
			http.Error(w, "Feed not found", http.StatusNotFound)
			return
		}

		feed, ok := app.conf.Feeds[name]
		if !ok {
			log.Warnf("feed does not exist %s", name)
			http.Error(w, "Feed not found", http.StatusNotFound)
			return
		}

		fileInfo, err := os.Stat(fn)
		if err != nil {
			log.WithError(err).Error("os.Stat() error")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
		w.Header().Set("Last-Modified", fileInfo.ModTime().Format(http.TimeFormat))

		if r.Method == http.MethodHead {
			return
		}

		ctx := map[string]string{
			"Name":         feed.Name,
			"URL":          fmt.Sprintf("%s/%s/twtxt.txt", app.conf.BaseURL, feed.Name),
			"Type":         feed.Type,
			"Source":       feed.URI,
			"Avatar":       feed.Avatar,
			"Description":  feed.Description,
			"LastModified": fileInfo.ModTime().UTC().Format(time.RFC3339),

			"SoftwareVersion": FullVersion(),
		}

		preamble, err := RenderPlainText(preambleTemplate, ctx)
		if err != nil {
			log.WithError(err).Warn("error rendering twtxt preamble")
		}

		f, err := os.Open(fn)
		if err != nil {
			log.WithError(err).Error("error opening feed")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", int64(len(preamble))+fileInfo.Size()))
		w.Header().Set("Last-Modified", fileInfo.ModTime().UTC().Format(http.TimeFormat))

		mrs := ioutil.NewMultiReadSeeker(strings.NewReader(preamble), f)
		http.ServeContent(w, r, "", fileInfo.ModTime(), mrs)
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func (app *App) AvatarHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead || r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, no-cache, must-revalidate")

		vars := mux.Vars(r)

		name := vars["name"]
		if name == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		fn := filepath.Join(app.conf.DataDir, fmt.Sprintf("%s.txt", name))
		if !Exists(fn) {
			log.Warnf("feed does not exist %s", name)
			http.Error(w, "Feed not found", http.StatusNotFound)
			return
		}

		fn = filepath.Join(app.conf.DataDir, fmt.Sprintf("%s.png", name))
		if fileInfo, err := os.Stat(fn); err == nil {
			etag := fmt.Sprintf("W/\"%s-%s\"", r.RequestURI, fileInfo.ModTime().Format(time.RFC3339))

			if match := r.Header.Get("If-None-Match"); match != "" {
				if strings.Contains(match, etag) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}

			w.Header().Set("Etag", etag)
			if r.Method == http.MethodHead {
				return
			}

			f, err := os.Open(fn)
			if err != nil {
				log.WithError(err).Error("error opening avatar file")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			defer f.Close()

			fileInfo, err := os.Stat(fn)
			if err != nil {
				log.WithError(err).Error("os.Stat() error")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

			if r.Method == http.MethodHead {
				return
			}

			if _, err := io.Copy(w, f); err != nil {
				log.WithError(err).Error("error writing avatar response")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			return
		}

		etag := fmt.Sprintf("W/\"%s\"", r.RequestURI)

		if match := r.Header.Get("If-None-Match"); match != "" {
			if strings.Contains(match, etag) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		w.Header().Set("Etag", etag)

		buf := bytes.Buffer{}
		img := cameron.Identicon([]byte(name), avatarResolution, 12)
		png.Encode(&buf, img)

		w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))

		if r.Method == http.MethodHead {
			return
		}

		w.Write(buf.Bytes())
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func (app *App) WeAreFeedsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead || r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/plain")

		if r.Method == http.MethodHead {
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		for _, feed := range app.GetFeeds() {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", feed.Name, feed.URI, feed.Avatar, feed.Description)
		}
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

func (app *App) FeedsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead || r.Method == http.MethodGet {
		if accept.PreferredContentTypeLike(r.Header, "text/plain") == "text/plain" {
			app.WeAreFeedsHandler(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")

		ctx := struct {
			Title string
			Feeds []Feed
		}{
			Title: "Available twtxt feeds",
			Feeds: app.GetFeeds(),
		}

		if r.Method == http.MethodHead {
			return
		}

		if err := render("feeds", feedsTemplate, ctx, w); err != nil {
			log.WithError(err).Error("error rendering feeds template")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}
