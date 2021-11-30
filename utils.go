package main

import (
	"encoding/base32"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	// Blank import so we can handle image/*
	_ "image/gif"
	_ "image/jpeg"

	"github.com/h2non/filetype"
	"github.com/nfnt/resize"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/blake2b"
)

var (
	ErrInvalidImage = errors.New("error: invalid image")
)

// CleanTwt cleans a twt's text, replacing new lines with spaces and
// stripping surrounding spaces.
func CleanTwt(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n", "\u2028")
	return text
}

func RotateFile(fn string) error {
	now := time.Now().Unix()
	newFn := fmt.Sprintf("%s.%d", fn, now)
	if err := os.Rename(fn, newFn); err != nil {
		log.WithError(err).Errorf("error renaming %s", fn)
		return fmt.Errorf("error renaming %s to %s: %w", fn, newFn)
	}
	return os.WriteFile(fn, nil, 0644)
}

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

	tf, err := ioutil.TempFile("", "feeds-*")
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

	fn := filepath.Join(conf.DataDir, filename)

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

func URLForFeed(conf *Config, name string) string {
	return fmt.Sprintf(
		"%s/%s/twtxt.txt",
		strings.TrimSuffix(conf.BaseURL, "/"),
		name,
	)
}

func WalkMatch(root, pattern string) ([]string, error) {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func BaseWithoutExt(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func FastHash(data []byte) string {
	sum := blake2b.Sum256(data)

	// Base32 is URL-safe, unlike Base64, and shorter than hex.
	encoding := base32.StdEncoding.WithPadding(base32.NoPadding)
	hash := strings.ToLower(encoding.EncodeToString(sum[:]))

	return hash
}

func FastHashString(s string) string {
	return FastHash([]byte(s))
}

func FastHashFile(fn string) (string, error) {
	data, err := os.ReadFile(fn)
	if err != nil {
		return "", err
	}
	return FastHash(data), nil
}
