package config

import (
	"gopkg.in/yaml.v3"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

type AppConfig struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DbConfig        `yaml:"database"`
	Messaging MessagingConfig `yaml:"mq,omitempty"`
}

type DbConfig struct {
	ConnectionString string `yaml:"connection-string"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
}

type MessagingConfig struct {
	Kafka KafkaMessagingConfig `yaml:"kafka,omitempty"`
}

type KafkaMessagingConfig struct {
	Readers []KafkaReaderConfig `yaml:"readers,omitempty"`
	Writers []KafkaWriterConfig `yaml:"writers,omitempty"`
}

type KafkaReaderConfig struct {
	Topic    string   `yaml:"topic"`
	Brokers  []string `yaml:"brokers"`
	GroupId  string   `yaml:"group-id"`
	MaxBytes int      `yaml:"max-bytes,omitempty"`
}

type KafkaWriterConfig struct {
	Topic   string   `yaml:"topic"`
	Brokers []string `yaml:"brokers"`
}

func MustLoad() *AppConfig {
	cfgPath := os.Getenv("GOSHRT_CONFIG_PATH")
	if cfgPath == "" {
		log.Fatalf("GOSHRT_CONFIG_PATH is not set")
	}

	cfgBaseName := os.Getenv("GOSHRT_CONFIG_NAME")
	if cfgBaseName == "" {
		log.Fatalf("GOSHRT_CONFIG_NAME is not set")
	}

	var cfgFileName string
	env := GetEnvironment()
	if env == "" {
		cfgFileName = cfgBaseName + ".yaml"
	} else {
		cfgFileName = strings.Join([]string{
			cfgBaseName,
			env,
			"yaml",
		}, ".")
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

	log.Printf("OK: Read config from %s", cfgFilePath)

	return &config
}
