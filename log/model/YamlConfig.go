package model

import "github.com/transsnet/vlog/log/kafka"

type LoggerConfig struct {
	EnableKafka bool               `yaml:"enable_kafka"`
	Base        *BaseLoggerConfig  `yaml:"base" validate:"required"`
	Kafka       *KafkaLoggerConfig `yaml:"kafka" validate:"omitempty"`
}

// 当业务应用不直接调用vlog打日志
// 而是封装一层后在打印,这时候需要配置caller_skip=1,其他情况下均不需要设置此值
type BaseLoggerConfig struct {
	LogPath     string `yaml:"log_path" validate:"gt=0"`
	LogLevel    string `yaml:"log_level"`
	ServiceName string `yaml:"service_name" validate:"gt=0"`
	CallerSkip  int    `yaml:"caller_skip" validate:"omitempty,gte=0,lte=2"`
}

type KafkaLoggerConfig struct {
	Client     *kafka.Client `yaml:"client" validate:"required"`
	InfoTopic  string        `yaml:"info_topic" validate:"omitempty"`
	ErrorTopic string        `yaml:"error_topic" validate:"gt=0"`
	Filter     []string      `yaml:"filter"`
}
