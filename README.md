# vlog
扩展zap日志，增加自己编写的kafka日志logger,在不影响原有日志输出情况下，同时往kafka里写

# example


```
package main

import (
	"github.com/transsnet/vlog/log"
	"github.com/transsnet/vlog/log/kafka"
	"time"
)

func main() {

	config := log.LoggerConfig{EnableKafkaLogger: true}
	config.BaseLoggerConfig = &log.BaseLoggerConfig{
		LogPath: "/tmp/log", Mode: "debug", ServiceName: "test-module",
	}
	config.KafkaLoggerConfig = &log.KafkaLoggerConfig{
		ErrorTopic: "palmads_error_log",
		InfoTopic:  "palmads_info_log",
	}
	config.KafkaLoggerConfig.KafkaClient = &kafka.KafkaClient{
		WorkerPool: &kafka.WorkerPool{
			Name:       "kafka-logger-pool",
			WorkerSize: 16,
			PoolSize:   1024 * 1024,
			Timeout:    4,
		},
		Hosts: []string{"172.31.19.94:32259"},
	}

	log.InitLog(&config)

	time1 := time.Now().UnixNano()
	// 十万次发送
	for a := 0; a < 100000; a++ {
		log.Info("Current best practice for ELK logging is to ship logs from hosts using Filebeat to " +
			"logstash where persistent queues are enabled. Filebeat supports structured (e.g. JSON)" +
			" and unstructured (e.g. log lines) log shipment.")
	}
	time2 := time.Now().UnixNano()

	println(time2 - time1)

}
```
