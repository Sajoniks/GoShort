package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"path"
)

type AppConfig struct {
	Server   ServerConfig `yaml:"server"`
	Database DbConfig     `yaml:"database"`
}

type DbConfig struct {
	ConnectionString string `yaml:"connection-string"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
}

func MustLoad() *AppConfig {
	cfgPath := os.Getenv("GOSHRT_CONFIG_PATH")
	if cfgPath == "" {
		log.Fatalf("GOSHRT_CONFIG_PATH is not set")
	}

	var cfgFileName string
	env := GetEnvironment()
	if env == "" {
		cfgFileName = "config.yaml"
	} else {
		cfgFileName = fmt.Sprintf("config.%s.yaml", env)
	}

	cfgFilePath := path.Join(cfgPath, cfgFileName)

	file, err := os.Open(cfgFilePath)
	if err != nil {
		log.Fatalf("Error opening config file: %v", err)
	}

	bs, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	var config AppConfig
	err = yaml.Unmarshal(bs, &config)

	if err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}

	return &config
}
