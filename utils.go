package main

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	// Blank import so we can handle image/*
	_ "image/gif"
	_ "image/jpeg"

	"github.com/h2non/filetype"
	"github.com/nfnt/resize"
	log "github.com/sirupsen/logrus"
)

var (
	validName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\- ]*$`)

	ErrInvalidName  = errors.New("error: invalid feed name")
	ErrNameTooLong  = errors.New("error: name is too long")
	ErrInvalidImage = errors.New("error: invalid image")
)

func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func IsImage(fn string) bool {
	f, err := os.Open(fn)
	if err != nil {
		log.WithError(err).Warnf("error opening file %s", fn)
		return false
	}
	defer f.Close()

	head := make([]byte, 261)
	if _, err := f.Read(head); err != nil {
		log.WithError(err).Warnf("error reading from file %s", fn)
		return false
	}

	if filetype.IsImage(head) {
		return true
	}

	return false
}

type ImageOptions struct {
	Resize  bool
	ResizeW int
	ResizeH int
}

func DownloadImage(conf *Config, url string, filename string, opts *ImageOptions) error {
	res, err := http.Get(url)
	if err != nil {
		log.WithError(err).Errorf("error downloading image from %s", url)
		return err
	}
	defer res.Body.Close()

	tf, err := ioutil.TempFile("", "rss2twtxt-*")
	if err != nil {
		log.WithError(err).Error("error creating temporary file")
		return err
	}
	defer tf.Close()

	if _, err := io.Copy(tf, res.Body); err != nil {
		log.WithError(err).Error("error writng temporary file")
		return err
	}

	if _, err := tf.Seek(0, io.SeekStart); err != nil {
		log.WithError(err).Error("error seeking temporary file")
		return err
	}

	if !IsImage(tf.Name()) {
		return ErrInvalidImage
	}

	if _, err := tf.Seek(0, io.SeekStart); err != nil {
		log.WithError(err).Error("error seeking temporary file")
		return err
	}

	if _, err := tf.Seek(0, io.SeekStart); err != nil {
		log.WithError(err).Error("error seeking temporary file")
		return err
	}

	img, _, err := image.Decode(tf)
	if err != nil {
		log.WithError(err).Error("jpeg.Decode failed")
		return err
	}

	newImg := img

	if opts != nil {
		if opts.Resize && (opts.ResizeW+opts.ResizeH > 0) && (opts.ResizeH > 0 || img.Bounds().Size().X > opts.ResizeW) {
			newImg = resize.Resize(uint(opts.ResizeW), uint(opts.ResizeH), img, resize.Lanczos3)
		}
	}

	fn := filepath.Join(conf.Root, filename)

	of, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.WithError(err).Error("error opening output file")
		return err
	}
	defer of.Close()

	// Encode uses a Writer, use a Buffer if you need the raw []byte
	if err := png.Encode(of, newImg); err != nil {
		log.WithError(err).Error("error reencoding image")
		return err
	}

	return nil
}

func AppendTwt(w io.Writer, text string, args ...interface{}) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("cowardly refusing to twt empty text, or only spaces")
	}

	// Support replacing/editing an existing Twt whilst preserving Created Timestamp
	// or posting a Twt with a custom Timestamp.
	now := time.Now().UTC()
	if len(args) == 1 {
		if t, ok := args[0].(time.Time); ok {
			now = t.UTC()
		}
	}

	line := fmt.Sprintf(
		"%s\t%s\n",
		now.Format(time.RFC3339),
		text,
	)

	if _, err := w.Write([]byte(line)); err != nil {
		return fmt.Errorf("error writing twt to writer: %w", err)
	}

	return nil
}

func FindClosestInt(target int, xs []int) int {
	n := sort.SearchInts(xs, target)
	if n >= len(xs) {
		return xs[len(xs)-1]
	}
	if xs[n]-target < target-xs[n-1] {
		n++
	}
	return xs[n-1]
}

func URLForFeed(conf *Config, name string) string {
	return fmt.Sprintf(
		"%s/%s/twtxt.txt",
		strings.TrimSuffix(conf.BaseURL, "/"),
		name,
	)
}
