package kafka

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/panjf2000/ants"
	"github.com/prometheus/common/log"
)

const (
	antsPoolDefaultName       = "kafka-work-pool"
	antsPoolDefaultSize       = 1024 * 1024
	antsPoolDefaultWorkerSize = 16
	antsPoolDefaultTimeout    = 4
)

var client *KafkaClient

type WorkerPool struct {
	Name        string
	WorkerSize  int
	PoolSize    int64
	Timeout     int
	pool        *ants.Pool
	submitFuncs chan func()
}

type KafkaClient struct {
	*WorkerPool
	Hosts         []string
	asyncProducer sarama.AsyncProducer
}

func InitKafkaClient(c *KafkaClient) {
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
		fmt.Println("kafka logger worker pool size >>>", config.PoolSize)
		config.start()
	} else {
		log.Fatal(err)
	}
}

func (workerPool *WorkerPool) start() {
	workerPool.submitFuncs = make(chan func(), workerPool.PoolSize)
	fmt.Println(">>WorkerPool start", workerPool.PoolSize, cap(workerPool.submitFuncs))
	go func() {
		for fun := range workerPool.submitFuncs {
			workerPool.pool.Submit(fun)
		}
	}()
}

func (workerPool *WorkerPool) AsyncSubmit(f func()) {
	workerPool.submitFuncs <- f
}

func initKafkaProducer(client *KafkaClient) {

	fmt.Println("start to init kafka msg queue..")

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
		errors := p.Errors()
		success := p.Successes()
		for {
			select {
			case err := <-errors:
				if err != nil {
					log.Fatal("sarama.AsyncProducer error:", err)
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

func (kafkaClient *KafkaClient) sendMsg(msg interface{}, topic string) error {

	if msg == nil {
		return nil
	}

	b, err := toJSONBytes(msg)

	if err != nil {
		return err
	}

	if kafkaClient.asyncProducer == nil {
		return errors.New("produce not init with hosts")
	}

	producerMsg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(b),
	}

	kafkaClient.append2WorkPool(producerMsg)
	return nil
}

func (kafkaClient *KafkaClient) append2WorkPool(producerMsg *sarama.ProducerMessage) {
	kafkaClient.AsyncSubmit(func() {
		kafkaClient.asyncProducer.Input() <- producerMsg
	})
}

func toJSONBytes(msg interface{}) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("message body is empty")
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	} else {
		return b, nil
	}
}
