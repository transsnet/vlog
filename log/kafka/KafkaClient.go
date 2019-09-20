package kafka

import (
	"errors"
	"github.com/Shopify/sarama"
	"github.com/panjf2000/ants"
	"log"
)

const (
	antsPoolDefaultName       = "kafka-work-pool"
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
	Name        string `yaml:"name" validate:"gt=0"`
	WorkerSize  int    `yaml:"worker_size" validate:"min=1"`
	PoolSize    int64  `yaml:"pool_size" validate:"min=100"`
	Timeout     int    `yaml:"timeout" validate:"min=1"`
	pool        *ants.Pool
	submitFuncs chan func()
}

func InitKafkaClient(c *Client) {
	client = c
	initKafkaWorkPool(client.WorkerPool)
	initKafkaProducer(client)
}

// 初始化工作线程池
func initKafkaWorkPool(config *WorkerPool) {
	var err error
	if config == nil {

		config = &WorkerPool{
			Name:       antsPoolDefaultName,
			WorkerSize: antsPoolDefaultWorkerSize,
			PoolSize:   antsPoolDefaultSize,
			Timeout:    antsPoolDefaultTimeout,
		}

	}

	if config.pool, err = ants.NewTimingPool(config.WorkerSize, config.Timeout); err == nil {
		log.Println("kafka logger worker pool size >>>", config.PoolSize)
		config.start()
	} else {
		log.Fatal(err)
	}
}

func (workerPool *WorkerPool) start() {
	workerPool.submitFuncs = make(chan func(), workerPool.PoolSize)
	log.Println(">>WorkerPool start", workerPool.PoolSize, cap(workerPool.submitFuncs))
	go func() {
		for fun := range workerPool.submitFuncs {
			if err := workerPool.pool.Submit(fun); err != nil {
				log.Println(err)
			}
		}
	}()
}

func (workerPool *WorkerPool) AsyncSubmit(f func()) {
	workerPool.submitFuncs <- f
}

func initKafkaProducer(client *Client) {

	log.Println("start to init kafka msg queue..")

	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.Return.Successes = true
	config.Producer.Retry.Max = 2
	config.ChannelBufferSize = 10240
	producer, err := sarama.NewAsyncProducer(client.Hosts, config)

	if err != nil {
		log.Fatal(err, client.Hosts)
	}

	go func(p sarama.AsyncProducer) {
		_errors := p.Errors()
		success := p.Successes()
		for {
			select {
			case err := <-_errors:
				if err != nil {
					log.Println("sarama.AsyncProducer error:", err)
				}
			case <-success:
			}
		}
	}(producer)

	if err != nil {
		log.Fatal(err)
	}

	client.asyncProducer = producer
}

func (kafkaClient *Client) sendMsg(msg []byte, topic string) error {

	if len(msg) == 0 {
		return nil
	}

	if kafkaClient.asyncProducer == nil {
		return errors.New("produce not init with hosts")
	}

	producerMsg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(msg),
	}

	kafkaClient.append2WorkPool(producerMsg)
	return nil
}

func (kafkaClient *Client) append2WorkPool(producerMsg *sarama.ProducerMessage) {
	kafkaClient.AsyncSubmit(func() {
		kafkaClient.asyncProducer.Input() <- producerMsg
	})
}
