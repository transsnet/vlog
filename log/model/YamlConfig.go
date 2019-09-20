package model

import "github.com/transsnet/vlog/log/kafka"

type LoggerConfig struct {
	EnableKafka bool               `yaml:"enable_kafka, omitempty"`
	Base        *BaseLoggerConfig  `yaml:"base" validate:"required"`
	Kafka       *KafkaLoggerConfig `yaml:"kafka, omitempty" validate:"omitempty"`
}

type BaseLoggerConfig struct {
	LogPath     string `yaml:"log_path" validate:"gt=0"`
	ServiceName string `yaml:"service_name" validate:"gt=0"`
}

type KafkaLoggerConfig struct {
	Client     *kafka.Client `yaml:"client" validate:"required"`
	InfoTopic  string        `yaml:"info_topic" validate:"gt=0"`
	ErrorTopic string        `yaml:"error_topic" validate:"gt=0"`
}