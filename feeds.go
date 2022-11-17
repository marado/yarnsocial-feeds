package main

import (
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

const (
	FeedTypeRSS = "rss"
	FeedTypeBot = "bot"
)

// Feed ...
type Feed struct {
	Name string
	URI  string
	Type string

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
	markdownRunes := []rune(markdown)
	if len(markdownRunes) > max {
		return fmt.Sprintf("%s ...", string(markdownRunes[:max]))
	}
	return markdown
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

// ValidateRSSFeed validates an RSS/Atom feed given a `uri` and returns a `Feed` object
// on success or a zero-value `Feed` object and `error` on an error.
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

	var avatar string
	if feed.Image != nil && feed.Image.URL != "" {
		opts := &ImageOptions{
			Resize:  true,
			ResizeW: avatarResolution,
			ResizeH: avatarResolution,
		}

		fn := fmt.Sprintf("%s.png", slug.Make(feed.Title))

		if err := DownloadImage(conf, feed.Image.URL, fn, opts); err != nil {
			log.WithError(err).Warnf("error downloading feed image from %s", feed.Image.URL)
		} else {
			avatar = fmt.Sprintf("%s/%s/avatar.png", conf.BaseURL, name)
			if avatarHash, err := FastHashFile(fn); err == nil {
				avatar += "#" + avatarHash
			} else {
				log.WithError(err).Warnf("error updating avatar hash for %s", name)
			}
		}
	}

	return Feed{
		Name:        name,
		URI:         uri,
		Avatar:      avatar,
		Description: feed.Description,
		Type:        FeedTypeRSS,
	}, nil
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

// ValidateMastodonFeed validates a Mastodon handle given a `uri` (a Mastodon handle)
//
//	and returns a `Feed` object on success or a zero-value `Feed` object and `error`
//
// on an error.
func ValidateMastodonFeed(conf *Config, handle string) (Feed, error) {
	user, server, err := ParseMastodonHandle(handle)
	if err != nil {
		return Feed{}, fmt.Errorf("error parsing Mastodon Handle %q: %w", handle, err)
	}

	rssURI := fmt.Sprintf("https://%s/@%s.rss", server, user)

	feed, err := TestRSSFeed(rssURI)
	if err != nil {
		return Feed{}, fmt.Errorf("error: invalid Mastodon RSS URI %q", rssURI)
	}

	name := handle

	var avatar string
	if feed.Image != nil && feed.Image.URL != "" {
		opts := &ImageOptions{
			Resize:  true,
			ResizeW: avatarResolution,
			ResizeH: avatarResolution,
		}

		fn := fmt.Sprintf("%s.png", slug.Make(feed.Title))

		if err := DownloadImage(conf, feed.Image.URL, fn, opts); err != nil {
			log.WithError(err).Warnf("error downloading feed image from %s", feed.Image.URL)
		} else {
			avatar = fmt.Sprintf("%s/%s/avatar.png", conf.BaseURL, name)
			if avatarHash, err := FastHashFile(fn); err == nil {
				avatar += "#" + avatarHash
			} else {
				log.WithError(err).Warnf("error updating avatar hash for %s", name)
			}
		}
	}

	return Feed{
		Name:        name,
		URI:         rssURI,
		Avatar:      avatar,
		Description: feed.Description,
		Type:        FeedTypeRSS,
	}, nil
}

func ParseMastodonHandle(handle string) (string, string, error) {
	tokens := strings.Split(handle, "@")
	if len(tokens) == 3 {
		return tokens[1], tokens[2], nil
	} else if len(tokens) == 2 {
		return tokens[0], tokens[1], nil
	} else {
		return "", "", fmt.Errorf("error: expected 2 or 3 tokens but got %d", len(tokens))
	}
}
