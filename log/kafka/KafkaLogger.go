package kafka

import (
	"go.uber.org/zap/zapcore"
	"log"
)

func New(topic string, filter []string) zapcore.WriteSyncer {
	return &kafkaLogger{Topic: topic, Filter: filter}
}

type kafkaLogger struct {
	Topic  string
	Filter []string
}

func (logger *kafkaLogger) Write(p []byte) (n int, err error) {
	return logger.send(p)
}

func (logger *kafkaLogger) Sync() error {
	return nil
}

// 实际发送日志到kafka
func (logger *kafkaLogger) send(p []byte) (n int, err error) {
	if client != nil {
		if err := client.sendMsg(p, logger.Topic, logger.Filter); err != nil {
			log.Println(err)
		}
	}
	return len(p), nil
}
