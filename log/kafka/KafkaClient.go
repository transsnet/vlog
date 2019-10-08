package kafka

import (
	"encoding/json"
	"errors"
	"github.com/Shopify/sarama"
	"github.com/panjf2000/ants"
	"log"
	"strings"
)

const (
	antsPoolDefaultSize       = 1024 * 1024
	antsPoolDefaultWorkerSize = 16
	antsPoolDefaultTimeout    = 4
)

var client *Client

type Client struct {
	*WorkerPool   `yaml:"worker_pool" validate:"omitempty"`
	Hosts         []string `yaml:"hosts" validate:"gt=0"`
	asyncProducer sarama.AsyncProducer
}

// 其中timeout的单位是秒
type WorkerPool struct {
	WorkerSize  int   `yaml:"worker_size" validate:"min=1"`
	PoolSize    int64 `yaml:"pool_size" validate:"min=100"`
	Timeout     int   `yaml:"timeout" validate:"min=1"`
	pool        *ants.Pool
	submitFuncs chan func()
}

//  初始化kafka客户端
func InitKafkaClient(c *Client) {
	client = c
	initKafkaWorkPool(client.WorkerPool)
	initKafkaProducer(client)
}

// 初始化工作线程池
func initKafkaWorkPool(config *WorkerPool) {

	if config == nil {
		config = &WorkerPool{
			WorkerSize: antsPoolDefaultWorkerSize,
			PoolSize:   antsPoolDefaultSize,
			Timeout:    antsPoolDefaultTimeout,
		}
	}

	var err error
	if config.pool, err = ants.NewTimingPool(config.WorkerSize, config.Timeout); err == nil {
		log.Println("Kafka logger worker pool size >>>", config.PoolSize)
		config.start()
	} else {
		log.Fatal(err)
	}
}

// 开启线程池
func (workerPool *WorkerPool) start() {
	workerPool.submitFuncs = make(chan func(), workerPool.PoolSize)
	log.Println(">>Kafka logger workerpool start", workerPool.PoolSize, cap(workerPool.submitFuncs))
	go func() {
		for fun := range workerPool.submitFuncs {
			if err := workerPool.pool.Submit(fun); err != nil {
				log.Println(err)
			}
		}
	}()
}

// 异步提交
func (workerPool *WorkerPool) AsyncSubmit(f func()) {
	workerPool.submitFuncs <- f
}

// 初始化kafka生产者
func initKafkaProducer(client *Client) {

	log.Println("Kafka logger start to init kafka msg queue ...")

	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.Return.Successes = true
	config.Producer.Retry.Max = 2
	config.ChannelBufferSize = 1024 * 10
	config.Version = sarama.V2_0_0_0
	producer, err := sarama.NewAsyncProducer(client.Hosts, config)

	if err != nil {
		log.Fatal(err, client.Hosts)
		return
	} else {
		client.asyncProducer = producer
	}

	go func(p sarama.AsyncProducer) {
		_errors := p.Errors()
		success := p.Successes()
		for {
			select {
			case err := <-_errors:
				if err != nil {
					log.Println("Kafka logger sarama.AsyncProducer error:", err)
				}
			case <-success:
			}
		}
	}(client.asyncProducer)
}

// 实际发送消息，消息体为json encoded
func (kafkaClient *Client) sendMsg(msg []byte, topic string, filter []string) error {

	if len(msg) == 0 {
		return nil
	}

	if kafkaClient.asyncProducer == nil {
		return errors.New("Kafka logger produce not init with hosts")
	}

	// 这里经测试，需要进行内存拷贝才行
	contentStr := string(msg)
	contentBytes := []byte(contentStr)

	producerMsg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(contentBytes),
	}

	kafkaClient.append2WorkPool(producerMsg, filter, contentStr)
	return nil
}

func (kafkaClient *Client) append2WorkPool(producerMsg *sarama.ProducerMessage, filter []string,
	content string) {
	kafkaClient.AsyncSubmit(func() {
		needSubmit := true

		if len(filter) > 0 {
			var mc MsgContent
			if err := json.Unmarshal([]byte(content), &mc); err != nil {
				log.Println("Kafka logger filter error when unmarshal msg content", err)
			} else {
				for i := range filter {
					if mc.Msg == "" || filter[i] == "" {
						continue
					}

					if "["+filter[i]+"]" == mc.Msg || filter[i] == mc.Msg || strings.Contains(mc.Msg, filter[i]) {
						needSubmit = false
						break
					}
				}
			}
		}

		if needSubmit {
			kafkaClient.asyncProducer.Input() <- producerMsg
		}
	})
}

type MsgContent struct {
	Msg string `json:"msg"`
}
