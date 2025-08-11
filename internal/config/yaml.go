package config

import (
	"github.com/fandasy/06.08.2025/pkg/e"
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

type Config struct {
	Logger          *Logger          `yaml:"logger"`
	Archiver        *Archiver        `yaml:"archiver"`
	LocalZipStorage *LocalZipStorage `yaml:"local_zip_storage"`
	HttpServer      *HttpServer      `yaml:"http_server"`
}

type Logger struct {
	Dir string `yaml:"dir"`
}

type Archiver struct {
	MaxTasks            uint32               `yaml:"max_tasks"`
	MaxObjects          int                  `yaml:"max_objects"`
	ValidExtension      []string             `yaml:"valid_extension"`
	ArchiveObjectGetter *ArchiveObjectGetter `yaml:"archive_object_getter"`
}

type ArchiveObjectGetter struct {
	ValidContentType []string `yaml:"valid_content_type"`
}

type LocalZipStorage struct {
	Dir string `yaml:"dir"`
}

type HttpServer struct {
	Addr        string        `yaml:"addr"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
}

func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(err)
	}

	return cfg
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, e.Wrap("failed to open config file", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)

	var cfg Config
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, e.Wrap("failed to parse config file", err)
	}

	return &cfg, nil
}
