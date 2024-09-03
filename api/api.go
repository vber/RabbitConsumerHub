package api

import (
	"database/sql"
	"go-rabbitmq-consumers/MQServer"
	"go-rabbitmq-consumers/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// UpdateRabbitMQConfig updates the RabbitMQ server configuration in the database
func UpdateRabbitMQConfig(database *sql.DB, config *MQServer.RabbitMQConfig) error {
	const FUNCNAME = "UpdateRabbitMQConfig"

	_, err := database.Exec(`UPDATE rabbitmq_config SET host = ?, port = ?, user = ?, password = ?, vhost = ? WHERE id = 1`,
		config.Host, config.Port, config.User, config.Password, config.Vhost)
	if err != nil {
		logger.E(FUNCNAME, "failed to update RabbitMQ configuration.", err.Error())
		return err
	}

	return nil
}

// FetchRabbitMQConfig fetches the RabbitMQ server configuration from the database
func FetchRabbitMQConfig(database *sql.DB) (*MQServer.RabbitMQConfig, error) {
	const FUNCNAME = "FetchRabbitMQConfig"

	row := database.QueryRow("SELECT host, port, user, password, vhost FROM rabbitmq_config WHERE id = 1")
	var config MQServer.RabbitMQConfig
	err := row.Scan(&config.Host, &config.Port, &config.User, &config.Password, &config.Vhost)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.E(FUNCNAME, "no RabbitMQ configuration found.")
			return nil, fiber.NewError(fiber.StatusNotFound, "no RabbitMQ configuration found")
		}
		logger.E(FUNCNAME, "failed to query RabbitMQ configuration from SQLite database.", err.Error())
		return nil, err
	}

	return &config, nil
}

// AddConsumer adds a new consumer to the database
func AddConsumer(database *sql.DB, consumer *MQServer.ConsumerParams) error {
	const FUNCNAME = "AddConsumer"

	_, err := database.Exec(`INSERT INTO consumers (name, status, queue_name, exchange_name, routing_key, death_queue_name, death_queue_bind_exchange, death_queue_bind_routing_key, death_queue_ttl, callback, retry_mode, queue_count) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		consumer.Name, consumer.Status, consumer.QueueName, consumer.ExchangeName, consumer.RoutingKey, consumer.DeathQueue.QueueName, consumer.DeathQueue.BindExchange, consumer.DeathQueue.BindRoutingKey, consumer.DeathQueue.TTL, consumer.Callback, consumer.RetryMode, consumer.QueueCount)
	if err != nil {
		logger.E(FUNCNAME, "failed to add consumer.", err.Error())
		return err
	}

	return nil
}

// EditConsumer updates an existing consumer in the database
func EditConsumer(database *sql.DB, consumer *MQServer.ConsumerParams) error {
	const FUNCNAME = "EditConsumer"

	_, err := database.Exec(`UPDATE consumers SET name = ?, status = ?, queue_name = ?, exchange_name = ?, routing_key = ?, death_queue_name = ?, death_queue_bind_exchange = ?, death_queue_bind_routing_key = ?, death_queue_ttl = ?, callback = ?, retry_mode = ?, queue_count = ? 
		WHERE id = ?`,
		consumer.Name, consumer.Status, consumer.QueueName, consumer.ExchangeName, consumer.RoutingKey, consumer.DeathQueue.QueueName, consumer.DeathQueue.BindExchange, consumer.DeathQueue.BindRoutingKey, consumer.DeathQueue.TTL, consumer.Callback, consumer.RetryMode, consumer.QueueCount, consumer.Id)
	if err != nil {
		logger.E(FUNCNAME, "failed to edit consumer.", err.Error())
		return err
	}

	return nil
}

// DeleteConsumer deletes a consumer from the database
func DeleteConsumer(database *sql.DB, consumerID string) error {
	const FUNCNAME = "DeleteConsumer"

	_, err := database.Exec(`DELETE FROM consumers WHERE id = ?`, consumerID)
	if err != nil {
		logger.E(FUNCNAME, "failed to delete consumer.", err.Error())
		return err
	}

	return nil
}

// EnableConsumer enables a consumer by setting its status to "running"
func EnableConsumer(database *sql.DB, consumerID string) error {
	const FUNCNAME = "EnableConsumer"

	_, err := database.Exec(`UPDATE consumers SET status = 'running' WHERE id = ?`, consumerID)
	if err != nil {
		logger.E(FUNCNAME, "failed to enable consumer.", err.Error())
		return err
	}

	return nil
}

// DisableConsumer disables a consumer by setting its status to "stopped"
func DisableConsumer(database *sql.DB, consumerID string) error {
	const FUNCNAME = "DisableConsumer"

	_, err := database.Exec(`UPDATE consumers SET status = 'stopped' WHERE id = ?`, consumerID)
	if err != nil {
		logger.E(FUNCNAME, "failed to disable consumer.", err.Error())
		return err
	}

	return nil
}

// RegisterRoutes registers the API routes with the Fiber app
func RegisterRoutes(app *fiber.App, db *sql.DB) {
	// Enable CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // You can specify allowed origins here
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	app.Get("/rabbitmq-config", func(c *fiber.Ctx) error {
		config, err := FetchRabbitMQConfig(db)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(config)
	})

	app.Put("/rabbitmq-config", func(c *fiber.Ctx) error {
		var config struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			User     string `json:"user"`
			Password string `json:"password"`
			Vhost    string `json:"vhost"`
		}
		if err := c.BodyParser(&config); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		rabbitMQConfig := MQServer.RabbitMQConfig{
			Host:     config.Host,
			Port:     config.Port,
			User:     config.User,
			Password: config.Password,
			Vhost:    config.Vhost,
		}
		if err := UpdateRabbitMQConfig(db, &rabbitMQConfig); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "RabbitMQ configuration updated successfully"})
	})

	app.Get("/consumers", func(c *fiber.Ctx) error {
		rows, err := db.Query("SELECT * FROM consumers")
		if err != nil {
			logger.E("GET /consumers", "Error querying database", err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database query error"})
		}
		defer rows.Close()

		var consumers []MQServer.ConsumerParams
		for rows.Next() {
			var consumer MQServer.ConsumerParams
			err := rows.Scan(
				&consumer.Id,
				&consumer.Name,
				&consumer.Status,
				&consumer.QueueName,
				&consumer.ExchangeName,
				&consumer.RoutingKey,
				&consumer.DeathQueue.QueueName,
				&consumer.DeathQueue.BindExchange,
				&consumer.DeathQueue.BindRoutingKey,
				&consumer.DeathQueue.TTL,
				&consumer.Callback,
				&consumer.RetryMode,
				&consumer.QueueCount,
			)
			if err != nil {
				logger.E("GET /consumers", "Error scanning row", err.Error())
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error scanning database row"})
			}
			consumers = append(consumers, consumer)
		}

		if err := rows.Err(); err != nil {
			logger.E("GET /consumers", "Error after iterating rows", err.Error())
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error after iterating rows"})
		}

		// If no consumers were found, return an empty array instead of null
		if len(consumers) == 0 {
			return c.JSON([]MQServer.ConsumerParams{})
		}

		return c.JSON(consumers)
	})

	app.Put("/consumers/:id", func(c *fiber.Ctx) error {
		var consumerData struct {
			Name         string `json:"name"`
			Status       string `json:"status"`
			QueueName    string `json:"queue_name"`
			ExchangeName string `json:"exchange_name"`
			RoutingKey   string `json:"routing_key"`
			Callback     string `json:"callback"`
			DeathQueue   struct {
				XDeathQueueName string `json:"x_death_queue_name"`
				BindExchange    string `json:"bind_exchange"`
				BindRoutingKey  string `json:"bind_routing_key"`
				XMessageTTL     string `json:"x_message_ttl"`
			} `json:"death_queue"`
			QueueCount uint64 `json:"queue_count"`
			RetryMode  string `json:"retry_mode"`
		}

		if err := c.BodyParser(&consumerData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		consumer := MQServer.ConsumerParams{
			Id:           c.Params("id"),
			Name:         consumerData.Name,
			Status:       consumerData.Status,
			QueueName:    consumerData.QueueName,
			ExchangeName: consumerData.ExchangeName,
			RoutingKey:   consumerData.RoutingKey,
			Callback:     consumerData.Callback,
			DeathQueue: MQServer.DeathQueueInfo{
				QueueName:      consumerData.DeathQueue.XDeathQueueName,
				BindExchange:   consumerData.DeathQueue.BindExchange,
				BindRoutingKey: consumerData.DeathQueue.BindRoutingKey,
				TTL:            consumerData.DeathQueue.XMessageTTL,
			},
			QueueCount: consumerData.QueueCount,
			RetryMode:  consumerData.RetryMode,
		}

		if err := EditConsumer(db, &consumer); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		ConsumerNotificationChan <- ConsumerNotification{Type: "updated", Consumer: consumer}

		return c.JSON(fiber.Map{"message": "Consumer updated successfully"})
	})

	app.Delete("/consumers/:id", func(c *fiber.Ctx) error {
		consumerID := c.Params("id")

		// Fetch the consumer before deleting
		consumer, err := FetchConsumer(db, consumerID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		if err := DeleteConsumer(db, consumerID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		ConsumerNotificationChan <- ConsumerNotification{Type: "deleted", Consumer: *consumer}

		return c.JSON(fiber.Map{"message": "Consumer deleted successfully"})
	})

	app.Put("/consumers/:id/enable", func(c *fiber.Ctx) error {
		consumerID := c.Params("id")
		if err := EnableConsumer(db, consumerID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Consumer enabled successfully"})
	})

	app.Put("/consumers/:id/disable", func(c *fiber.Ctx) error {
		consumerID := c.Params("id")
		if err := DisableConsumer(db, consumerID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Consumer disabled successfully"})
	})

	app.Post("/consumers", func(c *fiber.Ctx) error {
		var consumerData struct {
			Name         string `json:"name"`
			Status       string `json:"status"`
			QueueName    string `json:"queue_name"`
			ExchangeName string `json:"exchange_name"`
			RoutingKey   string `json:"routing_key"`
			Callback     string `json:"callback"`
			DeathQueue   struct {
				XDeathQueueName string `json:"x_death_queue_name"`
				BindExchange    string `json:"bind_exchange"`
				BindRoutingKey  string `json:"bind_routing_key"`
				XMessageTTL     string `json:"x_message_ttl"`
			} `json:"death_queue"`
			QueueCount uint64 `json:"queue_count"`
			RetryMode  string `json:"retry_mode"`
		}

		if err := c.BodyParser(&consumerData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		consumer := MQServer.ConsumerParams{
			Name:         consumerData.Name,
			Status:       consumerData.Status,
			QueueName:    consumerData.QueueName,
			ExchangeName: consumerData.ExchangeName,
			RoutingKey:   consumerData.RoutingKey,
			Callback:     consumerData.Callback,
			DeathQueue: MQServer.DeathQueueInfo{
				QueueName:      consumerData.DeathQueue.XDeathQueueName,
				BindExchange:   consumerData.DeathQueue.BindExchange,
				BindRoutingKey: consumerData.DeathQueue.BindRoutingKey,
				TTL:            consumerData.DeathQueue.XMessageTTL,
			},
			QueueCount: consumerData.QueueCount,
			RetryMode:  consumerData.RetryMode,
		}

		if err := AddConsumer(db, &consumer); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		ConsumerNotificationChan <- ConsumerNotification{Type: "added", Consumer: consumer}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Consumer created successfully"})
	})
}

// FetchConsumer fetches a single consumer from the database
func FetchConsumer(database *sql.DB, consumerID string) (*MQServer.ConsumerParams, error) {
	const FUNCNAME = "FetchConsumer"

	row := database.QueryRow("SELECT * FROM consumers WHERE id = ?", consumerID)
	var consumer MQServer.ConsumerParams
	err := row.Scan(
		&consumer.Id,
		&consumer.Name,
		&consumer.Status,
		&consumer.QueueName,
		&consumer.ExchangeName,
		&consumer.RoutingKey,
		&consumer.DeathQueue.QueueName,
		&consumer.DeathQueue.BindExchange,
		&consumer.DeathQueue.BindRoutingKey,
		&consumer.DeathQueue.TTL,
		&consumer.Callback,
		&consumer.RetryMode,
		&consumer.QueueCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.E(FUNCNAME, "no consumer found with ID: "+consumerID)
			return nil, fiber.NewError(fiber.StatusNotFound, "no consumer found with given ID")
		}
		logger.E(FUNCNAME, "failed to query consumer from SQLite database.", err.Error())
		return nil, err
	}

	return &consumer, nil
}

// ConsumerNotification is exported
type ConsumerNotification struct {
	Type     string                  `json:"type"`
	Consumer MQServer.ConsumerParams `json:"consumer"`
}

var ConsumerNotificationChan chan ConsumerNotification

func SetConsumerNotificationChan(ch chan ConsumerNotification) {
	ConsumerNotificationChan = ch
}
