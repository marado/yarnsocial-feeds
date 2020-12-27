package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/divan/num2words"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

var Jobs map[string]JobFactory

func init() {
	Jobs = map[string]JobFactory{
		"@every 5m":      NewUpdateFeedsJob,
		"0 0,30 * * * *": NewTikTokJob,
	}
}

type JobFactory func(conf *Config) cron.Job

type UpdateFeedsJob struct {
	conf *Config
}

func NewUpdateFeedsJob(conf *Config) cron.Job {
	return &UpdateFeedsJob{conf: conf}
}

func (job *UpdateFeedsJob) Run() {
	conf := job.conf
	for name, url := range conf.Feeds {
		if err := UpdateFeed(conf, name, url); err != nil {
			log.WithError(err).Errorf("error updating feed %s: %s", name, url)
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

	return &TikTokJob{
		conf:    conf,
		name:    name,
		url:     url,
		symbols: symbols,
	}
}

func (job *TikTokJob) Run() {
	conf := job.conf

	fn := filepath.Join(conf.Root, fmt.Sprintf("%s.txt", job.name))

	f, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.WithError(err).Error("error opening file for writing")
		return
	}
	defer f.Close()

	err = AppendTwt(f,
		fmt.Sprintf(`
I am %s an automated feed that twts every 30m with the current time (UTC)
`, job.url,
		),
		time.Unix(0, 0),
	)
	if err != nil {
		log.WithError(err).Error("error writing @tiktok feed")
		return
	}

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
