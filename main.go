package main

import (
	"fmt"
	"go-rabbitmq-consumers/MQServer"
	retry "go-rabbitmq-consumers/Retry"
	"go-rabbitmq-consumers/api"
	"go-rabbitmq-consumers/db"
	"go-rabbitmq-consumers/logger"
	"time"

	"github.com/gofiber/fiber/v2"
)

var (
	RabbitMQConf    *MQServer.RabbitMQConfig
	ConsumersConf   *MQServer.RabbitMQConsumers
	ConsumersPool   map[string]*MQServer.RabbitMQServer
	RetryServiceURL string // 重试服务URL
)

func init_config() {
	const FUNCNAME = "init_config"
	var err error

	// Initialize the database connection
	database, err := db.InitDB("./rch.db")
	if err != nil {
		logger.E(FUNCNAME, "failed to initialize database.", err.Error())
		panic(err)
	}
	defer database.Close()

	RabbitMQConf, err = db.FetchRabbitMQConfig(database)
	if err != nil {
		logger.E(FUNCNAME, "failed to fetch RabbitMQ configuration.", err.Error())
		panic(err)
	}

	ConsumersConf, err = db.FetchConsumersConfig(database)
	if err != nil {
		logger.E(FUNCNAME, "failed to fetch consumers configuration.", err.Error())
		panic(err)
	}

	RetryServiceURL, err = db.FetchRetryServiceURL(database)
	if err != nil {
		logger.E(FUNCNAME, "failed to fetch RetryServiceURL.", err.Error())
		panic(err)
	}
}

func init() {
	ConsumersPool = make(map[string]*MQServer.RabbitMQServer)
}

func start_consumer(consumer_config MQServer.ConsumerParams) {
	const FUNCNAME = "start_consumer"
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

	mq_server := MQServer.NewRabbitMQServer(RabbitMQConf)
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

	// Initialize Fiber app
	app := fiber.New()

	// Initialize the database connection
	database, err := db.InitDB("./rch.db")
	if err != nil {
		logger.E("main", "failed to initialize database.", err.Error())
		panic(err)
	}
	defer database.Close()

	// Register API routes
	api.RegisterRoutes(app, database)

	// Start Fiber app
	logger.I("main", "Starting API server on port 1981")
	if err := app.Listen(":1981"); err != nil {
		logger.E("main", "failed to start API server.", err.Error())
		panic(err)
	}

	for _, consumer := range ConsumersConf.Consumers {
		start_consumer(consumer)
	}

	forever := make(chan bool)
	<-forever
}
