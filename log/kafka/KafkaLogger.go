package kafka

import (
	"go.uber.org/zap/zapcore"
)

func New(topic string) zapcore.WriteSyncer {
	return &kafkaLogger{Topic: topic}
}

type kafkaLogger struct {
	Topic string
}

func (logger *kafkaLogger) Write(p []byte) (n int, err error) {
	return logger.send(p)
}

func (logger *kafkaLogger) Sync() error {
	return nil
}

func (logger *kafkaLogger) send(p []byte) (n int, err error) {
	if client != nil {
		client.sendMsg(string(p), logger.Topic)
	}
	return 0, nil
}
