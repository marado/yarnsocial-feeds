package main

import (
	"io/ioutil"

	"github.com/go-yaml/yaml"
)

type Config struct {
	Root    string
	BaseURL string
	MaxSize int64             // maximum feed size before rotating
	Feeds   map[string]string // name -> url

	path string // path to config file that was loaded used by .Save()
}

func (conf *Config) Parse(data []byte) error {
	return yaml.Unmarshal(data, conf)
}

func (conf *Config) Save() error {
	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}
	data = append([]byte("---\n"), data...)
	return ioutil.WriteFile(conf.path, data, 0644)
}

func LoadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	conf := &Config{}
	if err := conf.Parse(data); err != nil {
		return nil, err
	}
	conf.path = filename

	if conf.Feeds == nil {
		conf.Feeds = make(map[string]string)
	}

	return conf, nil
}
