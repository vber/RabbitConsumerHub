package main

import (
	"fmt"
	"go-rabbitmq-consumers/MQServer"
	"go-rabbitmq-consumers/api"
	"go-rabbitmq-consumers/db"
	"go-rabbitmq-consumers/logger"
	"sync"
	"time"

	"go-rabbitmq-consumers/models" // Add this import

	"github.com/gofiber/fiber/v2"
)

var (
	RabbitMQConf             *models.RabbitMQConfig
	ConsumersConf            *models.RabbitMQConsumers
	ConsumersPool            map[string]*MQServer.RabbitMQServer
	RetryServiceURL          string
	ConsumerNotificationChan chan api.ConsumerNotification
	ConsumersMutex           sync.RWMutex
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
	if RabbitMQConf == nil {
		logger.E(FUNCNAME, "RabbitMQConf is nil after fetching")
		panic("RabbitMQConf is nil")
	}
	logger.I(FUNCNAME, "RabbitMQConf successfully fetched")

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
	ConsumerNotificationChan = make(chan api.ConsumerNotification, 100)
}

func start_consumer(consumer_config models.ConsumerParams) {
	const FUNCNAME = "start_consumer"
	if RabbitMQConf == nil {
		logger.E(FUNCNAME, "RabbitMQConf is nil")
		return
	}

	if consumer_config.Status != "running" {
		logger.I(FUNCNAME, fmt.Sprintf("consumer would not start due to status=%s,queuename=%s", consumer_config.Status, consumer_config.QueueName))
		return
	}

	if consumer_config.DeathQueue.QueueName != "" {
		err := MQServer.CreateDeathQueue(RabbitMQConf, consumer_config.VHost, map[string]interface{}{
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
	if mq_server == nil {
		logger.E(FUNCNAME, "Failed to create RabbitMQServer instance")
		return
	}

	mq_server.DoError = func(queueData string, consumer *models.ConsumerParams) {

	}
	mq_server.DoSuccess = func(retry_id string) {}

	for {
		if mq_server.Connect(consumer_config.VHost) {
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

func handleConsumerNotifications() {
	for notification := range ConsumerNotificationChan {
		switch notification.Type {
		case "added":
			logger.I("main", fmt.Sprintf("add consumer. id:%s", notification.Consumer.Id))
			ConsumersMutex.Lock()
			start_consumer(notification.Consumer)
			ConsumersMutex.Unlock()
		case "updated":
			logger.I("main", fmt.Sprintf("update consumer. id:%s", notification.Consumer.Id))
			ConsumersMutex.Lock()

			if client, exists := ConsumersPool[notification.Consumer.Id]; exists {
				if notification.Consumer.Status == "stopped" {
					client.StopConsumer()
					delete(ConsumersPool, notification.Consumer.Id)
				}
			}

			if notification.Consumer.Status == "running" {
				start_consumer(notification.Consumer)
			}

			ConsumersMutex.Unlock()
		case "deleted":
			logger.I("main", fmt.Sprintf("delete consumer. id:%s", notification.Consumer.Id))
			ConsumersMutex.Lock()
			if client, exists := ConsumersPool[notification.Consumer.Id]; exists {
				logger.I("main", fmt.Sprintf("found delete consumer. id:%s", notification.Consumer.Id))
				if err := client.DeleteQueue(); err != nil {
					logger.E("main", fmt.Sprintf("Failed to delete queue %s: %s", client.Consumer.QueueName, err.Error()))
				}
				client.StopConsumer()
				delete(ConsumersPool, notification.Consumer.Id)
			}
			ConsumersMutex.Unlock()
		case "restarted":
			logger.I("main", fmt.Sprintf("restarting consumer. id:%s", notification.Consumer.Id))
			ConsumersMutex.Lock()
			if client, exists := ConsumersPool[notification.Consumer.Id]; exists {
				client.StopConsumer()
				delete(ConsumersPool, notification.Consumer.Id)
			}
			start_consumer(notification.Consumer)
			ConsumersMutex.Unlock()
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

	// Set the ConsumerNotificationChan
	api.SetConsumerNotificationChan(ConsumerNotificationChan)

	// Register API routes
	api.RegisterRoutes(app, database)

	go func() {
		for _, consumer := range ConsumersConf.Consumers {
			start_consumer(consumer)
		}
	}()

	go handleConsumerNotifications()

	// Start Fiber app
	logger.I("main", "Starting API server on port 1981")
	if err := app.Listen(":1981"); err != nil {
		logger.E("main", "failed to start API server.", err.Error())
		panic(err)
	}

	forever := make(chan bool)
	<-forever
}
