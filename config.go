package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-yaml/yaml"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Addr  string
	Debug bool

	DataDir     string
	BaseURL     string
	FeedsFile   string
	MaxFeedSize int64 // maximum feed size before rotating

	Feeds map[string]*Feed // name -> url
}

func (conf *Config) LoadFeeds() error {
	f, err := os.Open(conf.FeedsFile)
	if err != nil {
		log.WithError(err).Errorf("error opening feeds file %s", conf.FeedsFile)
		return fmt.Errorf("error opening feeds file %s: %w", conf.FeedsFile, err)
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.WithError(err).Errorf("error reading feeds file %s", conf.FeedsFile)
		return fmt.Errorf("error reading feeds file %s: %w", conf.FeedsFile, err)
	}

	if err := yaml.Unmarshal(data, conf.Feeds); err != nil {
		log.WithError(err).Errorf("error parsing feeds file %s", conf.FeedsFile)
		return fmt.Errorf("error parsing feeds file %s: %w", conf.FeedsFile, err)
	}

	for _, feed := range conf.Feeds {
		fn := filepath.Join(conf.DataDir, fmt.Sprintf("%s.png", feed.Name))
		if !Exists(fn) {
			continue
		}

		u, err := url.Parse(fn)
		if err != nil {
			log.WithError(err).Errorf("error parsing avatar url for %s", feed.Name)
			continue
		}

		avatarURL := fmt.Sprintf("%s/%s/avatar.png", conf.BaseURL, feed.Name)

		avatarHash, err := FastHashFile(fn)
		if err != nil {
			log.WithError(err).Errorf("error computing avatar hash for %s", feed.Name)
			fileInfo, err := os.Stat(fn)
			if err != nil {
				log.WithError(err).Errorf("error computing avatar stat for %s", feed.Name)
				continue
			}
			avatarURL += fmt.Sprintf("#%s", fileInfo.ModTime())
		} else {
			if u.Fragment != avatarHash {
				avatarURL += "#" + avatarHash
			}
		}
		feed.Avatar = avatarURL
	}

	return nil
}

func (conf *Config) SaveFeeds() error {
	f, err := os.OpenFile(conf.FeedsFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.WithError(err).Errorf("error opening feeds file %s", conf.FeedsFile)
		return fmt.Errorf("error opening feeds file %s: %w", conf.FeedsFile, err)
	}
	defer f.Close()

	data, err := yaml.Marshal(conf.Feeds)
	if err != nil {
		log.WithError(err).Errorf("error serializing feeds")
		return fmt.Errorf("error serializing feeds: %w", err)
	}

	data = append([]byte("---\n"), data...)

	if _, err := f.Write(data); err != nil {
		log.WithError(err).Errorf("error writing feeds file %s", conf.FeedsFile)
		return fmt.Errorf("error writing feeds file %s: %w", conf.FeedsFile, err)
	}

	return nil
}
