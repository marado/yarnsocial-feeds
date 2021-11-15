package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/andyleap/microformats"
	"github.com/gosimple/slug"
	"github.com/mmcdole/gofeed"
	twitterscraper "github.com/n0madic/twitter-scraper"
	log "github.com/sirupsen/logrus"
)

const (
	avatarResolution = 60 // 60x60 px
	twtxtTemplate    = "%s\t%s ⌘ [Read more](%s)\n"
	maxTwtLength     = 576
	maxTweets        = 10
)

var (
	ErrNoSuitableFeedsFound = errors.New("error: no suitable RSS or Atom feeds found")
)

// Feed ...
type Feed struct {
	Name string
	URI  string

	Avatar      string
	Description string

	LastModified string
}

func ProcessFeedContent(title, desc string, max int) string {
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(desc)
	if err != nil {
		log.WithError(err).Warnf("error converting content to html")
		return fmt.Sprintf("%s: %s", title, err)
	}
	markdown = CleanTwt(fmt.Sprintf("**%s**\n%s", title, markdown))
	if len(markdown) > max {
		return fmt.Sprintf("%s ...", markdown[:max])
	}
	return markdown
}

func TestTwitterFeed(handle string) error {
	count := 0
	for tweet := range twitterscraper.WithReplies(false).GetTweets(context.Background(), handle, maxTweets) {
		if tweet.Error != nil {
			return fmt.Errorf("error scraping tweets from %s: %w", handle, tweet.Error)
		}

		if tweet.IsRetweet {
			continue
		}
		count++
	}

	if count == 0 {
		log.WithField("handle", handle).WithField("handle", handle).Warn("empty or bad twitter handle")
	}

	return nil
}

func TestRSSFeed(uri string) (*gofeed.Feed, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(uri)
	if err != nil {
		return nil, err
	}

	return feed, nil
}

func FindRSSOrAtomAlternate(alts []*microformats.AlternateRel) string {
	for _, alt := range alts {
		switch alt.Type {
		case "application/atom+xml", "application/rss+xml":
			return alt.URL
		}
	}
	return ""
}

func FindRSSFeed(uri string) (*gofeed.Feed, string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, "", err
	}

	res, err := http.Get(u.String())
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	p := microformats.New()
	data := p.Parse(res.Body, u)

	altMap := make(map[string]string)
	for _, alt := range data.Alternates {
		altMap[alt.Type] = alt.URL
	}

	feedURI := altMap["application/atom+xml"]

	if feedURI == "" {
		feedURI = FindRSSOrAtomAlternate(data.Alternates)
	}

	if feedURI == "" {
		return nil, "", ErrNoSuitableFeedsFound
	}

	feed, err := TestRSSFeed(feedURI)
	if err != nil {
		return nil, "", err
	}

	return feed, feedURI, nil
}

// ValidateTwitterFeed ...
func ValidateTwitterFeed(conf *Config, handle string) (Feed, error) {
	err := TestTwitterFeed(handle)
	if err != nil {
		log.WithError(err).Warnf("invalid twitter feed %s", handle)
	}

	name := fmt.Sprintf("twitter-%s", handle)
	uri := fmt.Sprintf("twitter://%s", handle)

	opts := &ImageOptions{
		Resize:  true,
		ResizeW: avatarResolution,
		ResizeH: avatarResolution,
	}

	profile, err := twitterscraper.GetProfile(handle)
	if err != nil {
		log.WithError(err).Warnf("error retrieving twitter profile for %s", handle)
	}

	filename := fmt.Sprintf("%s.png", name)

	if err := DownloadImage(conf, profile.Avatar, filename, opts); err != nil {
		log.WithError(err).Warnf("error downloading feed image from %s", profile.Avatar)
	}

	return Feed{Name: name, URI: uri}, nil
}
// ValidateFeed ...
func ValidateRSSFeed(conf *Config, uri string) (Feed, error) {
	feed, err := TestRSSFeed(uri)
	if err != nil {
		log.WithError(err).Warnf("invalid rss feed %s", uri)
	}

	if feed == nil {
		feed, uri, err = FindRSSFeed(uri)
		if err != nil {
			log.WithError(err).Errorf("no rss feeds found on %s", uri)
			return Feed{}, err
		}
	}

	name := slug.Make(feed.Title)

	if feed.Image != nil && feed.Image.URL != "" {
		opts := &ImageOptions{
			Resize:  true,
			ResizeW: avatarResolution,
			ResizeH: avatarResolution,
		}

		filename := fmt.Sprintf("%s.png", name)

		if err := DownloadImage(conf, feed.Image.URL, filename, opts); err != nil {
			log.WithError(err).Warnf("error downloading feed image from %s", feed.Image.URL)
		}
	}

	return Feed{Name: name, URI: uri}, nil
}

// Code borrowed from https://github.com/n0madic/twitter2rss
// With permission from the author: https://github.com/n0madic/twitter2rss/issues/3
func UpdateTwitterFeed(conf *Config, name, handle string) error {
	var lastModified = time.Time{}

	fn := filepath.Join(conf.DataDir, fmt.Sprintf("%s.txt", name))

	stat, err := os.Stat(fn)
	if err == nil {
		lastModified = stat.ModTime()
	}

	f, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	old, new := 0, 0
	for tweet := range twitterscraper.WithReplies(false).GetTweets(context.Background(), handle, maxTweets) {
		if tweet.Error != nil {
			return fmt.Errorf("error scraping tweets from %s: %w", handle, tweet.Error)
		}

		if tweet.IsRetweet {
			continue
		}

		if tweet.TimeParsed.After(lastModified) {
			var title string

			titleSplit := strings.FieldsFunc(tweet.Text, func(r rune) bool {
				return r == '\n' || r == '!' || r == '?' || r == ':' || r == '<' || r == '.' || r == ','
			})
			if len(titleSplit) > 0 {
				if strings.HasPrefix(titleSplit[0], "a href") || strings.HasPrefix(titleSplit[0], "http") {
					title = "link"
				} else {
					title = titleSplit[0]
				}
			}
			title = strings.TrimSuffix(title, "https")
			title = strings.TrimSpace(title)

			text := fmt.Sprintf(
				twtxtTemplate,
				tweet.TimeParsed.Format(time.RFC3339),
				ProcessFeedContent(title, tweet.HTML, maxTwtLength-len(tweet.PermanentURL)),
				tweet.PermanentURL,
			)
			_, err := f.WriteString(text)
			if err != nil {
				return err
			}

		} else {
			old++
		}
	}

	opts := &ImageOptions{
		Resize:  true,
		ResizeW: avatarResolution,
		ResizeH: avatarResolution,
	}

	profile, err := twitterscraper.GetProfile(handle)
	if err != nil {
		log.WithError(err).Warnf("error retrieving twitter profile for %s", handle)
	}

	filename := fmt.Sprintf("%s.png", name)

	if err := DownloadImage(conf, profile.Avatar, filename, opts); err != nil {
		log.WithError(err).Warnf("error downloading feed image from %s", profile.Avatar)
	}

	if (old + new) == 0 {
		log.WithField("name", name).WithField("handle", handle).Warn("empty or bad twitter handle")
	}

	return nil
}

func UpdateRSSFeed(conf *Config, name, url string) error {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return err
	}

	avatarFile := filepath.Join(conf.DataDir, fmt.Sprintf("%s.png", name))
	if feed.Image != nil && feed.Image.URL != "" && !Exists(avatarFile) {
		opts := &ImageOptions{
			Resize:  true,
			ResizeW: avatarResolution,
			ResizeH: avatarResolution,
		}

		filename := fmt.Sprintf("%s.png", name)

		if err := DownloadImage(conf, feed.Image.URL, filename, opts); err != nil {
			log.WithError(err).Warnf("error downloading feed image from %s", feed.Image.URL)
		}
	}

	var lastModified = time.Time{}

	fn := filepath.Join(conf.DataDir, fmt.Sprintf("%s.txt", name))

	stat, err := os.Stat(fn)
	if err == nil {
		lastModified = stat.ModTime()
	}

	f, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	old, new := 0, 0
	for _, item := range feed.Items {
		if item.PublishedParsed == nil {
			continue
		}

		if item.PublishedParsed.After(lastModified) {
			new++
			text := fmt.Sprintf(
				twtxtTemplate,
				item.PublishedParsed.Format(time.RFC3339),
				ProcessFeedContent(item.Title, item.Description, maxTwtLength-len(item.Link)),
				item.Link,
			)
			_, err := f.WriteString(text)
			if err != nil {
				return err
			}
		} else {
			old++
		}
	}

	if (old + new) == 0 {
		log.WithField("name", name).WithField("url", url).Warn("empty or bad feed")
	}

	return nil
}
