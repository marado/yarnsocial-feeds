package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/divan/num2words"
	"github.com/dustin/go-humanize"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

// JobSpec ...
type JobSpec struct {
	Schedule string
	Factory  JobFactory
}

func NewJobSpec(schedule string, factory JobFactory) JobSpec {
	return JobSpec{schedule, factory}
}

var (
	Jobs        map[string]JobSpec
	StartupJobs map[string]JobSpec
)

func init() {
	Jobs = map[string]JobSpec{
		"RotateFeeds": NewJobSpec("@daily", NewRotateFeedsJob),
		"UpdateFeeds": NewJobSpec("@every 5m", NewUpdateFeedsJob),
		"TikTokBot":   NewJobSpec("0 0,30 * * * *", NewTikTokJob),
	}

	StartupJobs = map[string]JobSpec{
		"RotateFeeds": Jobs["RotateFeeds"],
	}
}

type JobFactory func(conf *Config) cron.Job

type RotateFeedsJob struct {
	conf *Config
}

func NewRotateFeedsJob(conf *Config) cron.Job {
	return &RotateFeedsJob{conf: conf}
}

func (job *RotateFeedsJob) Run() {
	conf := job.conf

	files, err := WalkMatch(conf.DataDir, "*.txt")
	if err != nil {
		log.WithError(err).Error("error reading feeds directory")
		return
	}

	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			log.WithError(err).Error("error getting feed size")
			continue
		}

		if stat.Size() > conf.MaxFeedSize {
			log.Infof(
				"rotating %s with size %s > %s",
				BaseWithoutExt(file),
				humanize.Bytes(uint64(stat.Size())),
				humanize.Bytes(uint64(conf.MaxFeedSize)),
			)

			if err := RotateFile(file); err != nil {
				log.WithError(err).Error("error rotating feed")
			}
		}
	}
}

type UpdateFeedsJob struct {
	conf *Config
}

func NewUpdateFeedsJob(conf *Config) cron.Job {
	return &UpdateFeedsJob{conf: conf}
}

func (job *UpdateFeedsJob) Run() {
	conf := job.conf
	for name, feed := range conf.Feeds {
		u, err := ParseURI(feed.URI)
		if err != nil {
			log.WithError(err).Errorf("error parsing feed %s: %s", name, feed.URI)
		} else {
			switch u.Type {
			case "rss", "http", "https":
				if err := UpdateRSSFeed(conf, name, feed.URI); err != nil {
					log.WithError(err).Errorf("error updating rss feed %s: %s", name, feed.URI)
				}
			case "twitter":
				if err := UpdateTwitterFeed(conf, name, u.Config); err != nil {
					log.WithError(err).Errorf("error updating twitter feed %s: %s", name, feed.URI)
				}
			default:
				log.Warnf("error unknown feed type %s: %s", name, feed.URI)
			}
		}
	}
}

type TikTokJob struct {
	conf    *Config
	name    string
	url     string
	symbols map[int]string
}

func NewTikTokJob(conf *Config) cron.Job {
	symbols := map[int]string{
		0: "🕛", 30: "🕧",
		100: "🕐", 130: "🕜",
		200: "🕑", 230: "🕝",
		300: "🕒", 330: "🕞",
		400: "🕓", 430: "🕟",
		500: "🕔", 530: "🕠",
		600: "🕕", 630: "🕡",
		700: "🕖", 730: "🕢",
		800: "🕗", 830: "🕣",
		900: "🕘", 930: "🕤",
		1000: "🕙", 1030: "🕥",
		1100: "🕚", 1130: "🕦",
		1200: "🕛", 1230: "🕧",
	}

	name := "tiktok"
	url := fmt.Sprintf("@<%s %s>", name, URLForFeed(conf, name))

	conf.Feeds[name] = &Feed{
		Name: name,
		Description: fmt.Sprintf(
			"I am @%s an automated feed that twts every 30m with the current time (UTC)",
			name,
		),
	}

	return &TikTokJob{
		conf:    conf,
		name:    name,
		url:     url,
		symbols: symbols,
	}
}

func (job *TikTokJob) Run() {
	conf := job.conf

	fn := filepath.Join(conf.DataDir, fmt.Sprintf("%s.txt", job.name))

	f, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.WithError(err).Error("error opening file for writing")
		return
	}
	defer f.Close()

	now := time.Now().UTC()

	hour := now.Hour() % 12
	min := now.Minute()

	var key int

	if hour == 0 {
		key = hour + min
	} else {
		key = (hour * 100) + min
	}
	sym := job.symbols[key]

	var clock string

	if hour == 0 {
		clock = "twelve"
	} else {
		clock = num2words.Convert(hour)
	}

	if min == 0 {
		clock += " o'clock"
	} else if min == 30 {
		clock += " thirty"
	} else {
		clock = fmt.Sprintf("%s past %s", num2words.Convert(min), clock)
	}

	if now.Hour() < 6 {
		clock += " in the morning 😴"
	} else if now.Hour() < 12 {
		clock += " 🌞"
	} else if now.Hour() < 18 {
		clock += " in the afternoon 🌅"
	} else {
		clock += " in the evening 🌛"
	}

	if err := AppendTwt(f, fmt.Sprintf("%s The time is now %s", sym, clock)); err != nil {
		log.WithError(err).Error("error writing @tiktok feed")
	}
}
