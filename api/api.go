package api

import (
	"database/sql"
	"fmt"
	"go-rabbitmq-consumers/logger"
	"go-rabbitmq-consumers/models"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/streadway/amqp"
)

// UpdateRabbitMQConfig updates the RabbitMQ server configuration in the database
func UpdateRabbitMQConfig(database *sql.DB, config *models.RabbitMQConfig) error {
	const FUNCNAME = "UpdateRabbitMQConfig"

	_, err := database.Exec(`UPDATE rabbitmq_config SET host = ?, port = ?, user = ?, password = ? WHERE id = 1`,
		config.Host, config.Port, config.User, config.Password)
	if err != nil {
		logger.E(FUNCNAME, "failed to update RabbitMQ configuration.", err.Error())
		return err
	}

	return nil
}

// FetchRabbitMQConfig fetches the RabbitMQ server configuration from the database
func FetchRabbitMQConfig(database *sql.DB) (*models.RabbitMQConfig, error) {
	const FUNCNAME = "FetchRabbitMQConfig"

	row := database.QueryRow("SELECT host, port, user, password FROM rabbitmq_config WHERE id = 1")
	var config models.RabbitMQConfig
	err := row.Scan(&config.Host, &config.Port, &config.User, &config.Password)
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

// AddConsumer adds a new consumer to the database and returns the new ID
func AddConsumer(database *sql.DB, consumer *models.ConsumerParams) (int64, error) {
	const FUNCNAME = "AddConsumer"

	result, err := database.Exec(`INSERT INTO consumers (name, status, queue_name, exchange_name, routing_key, death_queue_name, death_queue_bind_exchange, death_queue_bind_routing_key, death_queue_ttl, callback, retry_mode, queue_count, vhost) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		consumer.Name, consumer.Status, consumer.QueueName, consumer.ExchangeName, consumer.RoutingKey, consumer.DeathQueue.QueueName, consumer.DeathQueue.BindExchange, consumer.DeathQueue.BindRoutingKey, consumer.DeathQueue.TTL, consumer.Callback, consumer.RetryMode, consumer.QueueCount, consumer.VHost)
	if err != nil {
		logger.E(FUNCNAME, "failed to add consumer.", err.Error())
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		logger.E(FUNCNAME, "failed to get last insert ID.", err.Error())
		return 0, err
	}

	return id, nil
}

// EditConsumer updates an existing consumer in the database
func EditConsumer(database *sql.DB, consumer *models.ConsumerParams) error {
	const FUNCNAME = "EditConsumer"

	_, err := database.Exec(`UPDATE consumers SET name = ?, status = ?, queue_name = ?, exchange_name = ?, routing_key = ?, death_queue_name = ?, death_queue_bind_exchange = ?, death_queue_bind_routing_key = ?, death_queue_ttl = ?, callback = ?, retry_mode = ?, queue_count = ?, vhost = ? 
		WHERE id = ?`,
		consumer.Name, consumer.Status, consumer.QueueName, consumer.ExchangeName, consumer.RoutingKey, consumer.DeathQueue.QueueName, consumer.DeathQueue.BindExchange, consumer.DeathQueue.BindRoutingKey, consumer.DeathQueue.TTL, consumer.Callback, consumer.RetryMode, consumer.QueueCount, consumer.VHost, consumer.Id)
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
		}
		if err := c.BodyParser(&config); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		rabbitMQConfig := models.RabbitMQConfig{
			Host:     config.Host,
			Port:     config.Port,
			User:     config.User,
			Password: config.Password,
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

		var consumers []models.ConsumerParams
		for rows.Next() {
			var consumer models.ConsumerParams
			err := rows.Scan(
				&consumer.Id,
				&consumer.Name,
				&consumer.Status,
				&consumer.QueueName,
				&consumer.ExchangeName,
				&consumer.RoutingKey,
				&consumer.VHost,
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
			return c.JSON([]models.ConsumerParams{})
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
			Vhost      string `json:"vhost"`
		}

		if err := c.BodyParser(&consumerData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		consumer := models.ConsumerParams{
			Id:           c.Params("id"),
			Name:         consumerData.Name,
			Status:       consumerData.Status,
			QueueName:    consumerData.QueueName,
			ExchangeName: consumerData.ExchangeName,
			RoutingKey:   consumerData.RoutingKey,
			Callback:     consumerData.Callback,
			DeathQueue: models.DeathQueueInfo{
				QueueName:      consumerData.DeathQueue.XDeathQueueName,
				BindExchange:   consumerData.DeathQueue.BindExchange,
				BindRoutingKey: consumerData.DeathQueue.BindRoutingKey,
				TTL:            consumerData.DeathQueue.XMessageTTL,
			},
			QueueCount: consumerData.QueueCount,
			RetryMode:  consumerData.RetryMode,
			VHost:      consumerData.Vhost,
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
			Vhost      string `json:"vhost"`
		}

		if err := c.BodyParser(&consumerData); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		consumer := models.ConsumerParams{
			Name:         consumerData.Name,
			Status:       consumerData.Status,
			QueueName:    consumerData.QueueName,
			ExchangeName: consumerData.ExchangeName,
			RoutingKey:   consumerData.RoutingKey,
			Callback:     consumerData.Callback,
			DeathQueue: models.DeathQueueInfo{
				QueueName:      consumerData.DeathQueue.XDeathQueueName,
				BindExchange:   consumerData.DeathQueue.BindExchange,
				BindRoutingKey: consumerData.DeathQueue.BindRoutingKey,
				TTL:            consumerData.DeathQueue.XMessageTTL,
			},
			QueueCount: consumerData.QueueCount,
			RetryMode:  consumerData.RetryMode,
			VHost:      consumerData.Vhost,
		}

		id, err := AddConsumer(db, &consumer)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		consumer.Id = strconv.FormatInt(id, 10)
		ConsumerNotificationChan <- ConsumerNotification{Type: "added", Consumer: consumer}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "Consumer created successfully",
			"id":      id,
		})
	})

	app.Put("/consumers/:id/restart", func(c *fiber.Ctx) error {
		consumerID := c.Params("id")
		consumer, err := FetchConsumer(db, consumerID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		ConsumerNotificationChan <- ConsumerNotification{Type: "restarted", Consumer: *consumer}
		return c.JSON(fiber.Map{"message": "Consumer restarted successfully"})
	})

	app.Post("/test-rabbitmq-connection", func(c *fiber.Ctx) error {
		var config struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			User     string `json:"user"`
			Password string `json:"password"`
		}
		if err := c.BodyParser(&config); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}

		// Construct the AMQP URL
		amqpURL := fmt.Sprintf("amqp://%s:%s@%s:%d/", config.User, config.Password, config.Host, config.Port)

		// Try to establish a connection
		conn, err := amqp.Dial(amqpURL)
		if err != nil {
			logger.E("TestRabbitMQConnection", "Failed to connect to RabbitMQ", err.Error())
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to connect to RabbitMQ"})
		}
		defer conn.Close()

		return c.JSON(fiber.Map{"message": "Connection successful"})
	})
}

// FetchConsumer fetches a single consumer from the database
func FetchConsumer(database *sql.DB, consumerID string) (*models.ConsumerParams, error) {
	const FUNCNAME = "FetchConsumer"

	row := database.QueryRow("SELECT * FROM consumers WHERE id = ?", consumerID)
	var consumer models.ConsumerParams
	err := row.Scan(
		&consumer.Id,
		&consumer.Name,
		&consumer.Status,
		&consumer.QueueName,
		&consumer.ExchangeName,
		&consumer.RoutingKey,
		&consumer.VHost,
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
	Type     string                `json:"type"`
	Consumer models.ConsumerParams `json:"consumer"`
}

var ConsumerNotificationChan chan ConsumerNotification

func SetConsumerNotificationChan(ch chan ConsumerNotification) {
	ConsumerNotificationChan = ch
}
