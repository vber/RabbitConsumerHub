package main

import (
	"encoding/json"
	"fmt"
	"go-rabbitmq-consumers/MQServer"
	retry "go-rabbitmq-consumers/Retry"
	"go-rabbitmq-consumers/logger"
	"time"

	"github.com/vber/nacos/v2"
)

var (
	RabbitMQConf    *MQServer.RabbitMQConfig
	ConsumersConf   *MQServer.RabbitMQConsumers
	ConsumersPool   map[string]*MQServer.VinehooRabbitMQServer
	RetryServiceURL string // 重试服务URL
)

type MongoConfig struct {
	Master   string `json:"mongo_client_master"`
	Slave    string `json:"mongo_client_slave"`
	Port     int    `json:"mongo_port"`
	User     string `json:"mongo_auth"`
	Password string `json:"mongo_password"`
}

func init_config() {
	var (
		err    error
		config string
	)

	const (
		FUNCNAME = "init_config"
	)

	config, err = nacos.GetString("rabbitmq", "vinehoo.accounts", nil)
	if err != nil {
		logger.E(FUNCNAME, err.Error())
		panic(err)
	}
	if config == "" {
		logger.E(FUNCNAME, "rabbitmq configuration is empty!")
		panic("rabbitmq configuration is empty!")
	}

	err = json.Unmarshal([]byte(config), &RabbitMQConf)
	if err != nil {
		logger.E(FUNCNAME, err.Error())
		panic(err)
	}

	config, err = nacos.GetString("rabbitmq.consumers", "vinehoo.services", func(data *string, err error) {
		if err != nil {
			logger.E(FUNCNAME, "nacos listener error:", err.Error())
			return
		}
		logger.I(FUNCNAME, "nacos listener data:", *data)
		new_conf := MQServer.RabbitMQConsumers{}
		e := json.Unmarshal([]byte(*data), &new_conf)
		if e == nil {
			for _, item := range new_conf.Consumers {
				if ConsumersPool[item.Id] != nil {
					if item.Status == "stop" {
						s := ConsumersPool[item.Id]
						s.StopConsumer()
						delete(ConsumersPool, item.Id)
						logger.I("nacos listener", fmt.Sprintf("%s consumer(id:%s) stopped.", item.Name, item.Id))
					}
				} else {
					start_consumer(item)
				}
			}
		} else {
			logger.E("nacos listener", err.Error())
		}
	})

	if err != nil {
		panic(err)
	}
	if config == "" {
		logger.E(FUNCNAME, "consumers configuration are empty!")
		panic("consumers configuration are empty!")
	}

	err = json.Unmarshal([]byte(config), &ConsumersConf)
	if err != nil {
		logger.E(FUNCNAME, err.Error())
		panic(err)
	}

	RetryServiceURL, err = nacos.GetString("retry.url", "vinehoo.conf", func(data *string, err error) {
		if err == nil {
			RetryServiceURL = *data
		}
	})
	if err != nil {
		logger.E(FUNCNAME, "failed to get retry.url config.", err.Error())
		panic(err)
	}
}

func init() {
	ConsumersPool = make(map[string]*MQServer.VinehooRabbitMQServer)
}

func start_consumer(consumer_config MQServer.ConsumerParams) {
	const (
		FUNCNAME = "start_consumer"
	)
	if consumer_config.Status != "running" {
		logger.I(FUNCNAME, fmt.Sprintf("consumer would not start due to staus=%s,queuename=%s", consumer_config.Status, consumer_config.QueueName))
		return
	}

	if consumer_config.DeathQueue.QueueName != "" {
		err := MQServer.CreateDeathQueue(RabbitMQConf, map[string]interface{}{
			"x_death_queue_name":        consumer_config.DeathQueue.QueueName,
			"x_dead_letter_exchange":    consumer_config.ExchangeName,
			"x_dead_letter_routing_key": consumer_config.RoutingKey,
			"bind_exchange":             consumer_config.DeathQueue.BindExchange,
			"bind_routing_key":          consumer_config.DeathQueue.BindRoutingKey,
			"x_message_ttl":             consumer_config.DeathQueue.TTL,
		})
		if err != nil {
			logger.E(FUNCNAME, "create deathqueue failed.queuename:", consumer_config.QueueName, ",death_queuename:", consumer_config.DeathQueue.QueueName, ",err:", err.Error())
		} else {
			logger.I(FUNCNAME, "create deathqueue ok.queuename:", consumer_config.QueueName, ",death_queuename:", consumer_config.DeathQueue.QueueName)
		}
	}

	mq_server := MQServer.NewVinehooRabbitMQServer(RabbitMQConf)
	mq_server.DoError = func(queueData string, consumer *MQServer.ConsumerParams) {
		r := retry.RetryURL{
			QueueData: queueData,
			ReqURL:    consumer.Callback,
			RetryMode: consumer.RetryMode,
			RetryAPI:  &RetryServiceURL,
		}
		if err := r.RetryRequest(); err != nil {
			logger.E("Do Error:", err)
		}
	}
	mq_server.DoSuccess = func(retry_id string) {}

	for {
		if mq_server.Connect() {
			logger.I("main", "RabbitMQ server is connected.")
			break
		} else {
			logger.E("main", "RabbitMQ Server connected failed.")
			time.Sleep(3 * time.Second)
		}
	}

	if consumer_config.QueueCount == 0 {
		consumer_config.QueueCount = 1
	}

	for i := 0; i < int(consumer_config.QueueCount); i++ {
		err := mq_server.StartConsumer(&consumer_config)
		if err != nil {
			logger.E("main", fmt.Sprintf("failed to start consumer. id:%s, queue_name:%s, error:%s", consumer_config.Id, consumer_config.Name, err.Error()))
		} else {
			ConsumersPool[consumer_config.Id] = mq_server
			logger.I("main", fmt.Sprintf("start %s consumer ok. id:%s", consumer_config.Name, consumer_config.Id))
		}
	}
}

func main() {
	init_config()

	for _, consumer := range ConsumersConf.Consumers {
		start_consumer(consumer)
	}

	forever := make(chan bool)
	<-forever
}
