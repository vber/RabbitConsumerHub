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

	_, err := database.Exec(`UPDATE rabbitmq_config SET host = ?, port = ?, user = ?, password = ? WHERE id = 1`,
		config.Host, config.Port, config.User, config.Password)
	if err != nil {
		logger.E(FUNCNAME, "failed to update RabbitMQ configuration.", err.Error())
		return err
	}

	return nil
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

	app.Put("/rabbitmq/config", func(c *fiber.Ctx) error {
		var config MQServer.RabbitMQConfig
		if err := c.BodyParser(&config); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if err := UpdateRabbitMQConfig(db, &config); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "RabbitMQ configuration updated successfully"})
	})

	app.Get("/consumers", func(c *fiber.Ctx) error {
		rows, err := db.Query("SELECT * FROM consumers")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		defer rows.Close()

		var consumers []MQServer.ConsumerParams
		for rows.Next() {
			var consumer MQServer.ConsumerParams
			var deathQueueName, deathQueueBindExchange, deathQueueBindRoutingKey sql.NullString
			var deathQueueTTL string
			if err := rows.Scan(&consumer.Id, &consumer.Name, &consumer.Status, &consumer.QueueName, &consumer.ExchangeName, &consumer.RoutingKey, &deathQueueName, &deathQueueBindExchange, &deathQueueBindRoutingKey, &deathQueueTTL, &consumer.Callback, &consumer.RetryMode, &consumer.QueueCount); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
			consumer.DeathQueue.QueueName = deathQueueName.String
			consumer.DeathQueue.BindExchange = deathQueueBindExchange.String
			consumer.DeathQueue.BindRoutingKey = deathQueueBindRoutingKey.String
			consumers = append(consumers, consumer)
		}

		return c.JSON(consumers)
	})

	app.Put("/consumers/:id", func(c *fiber.Ctx) error {
		var consumer MQServer.ConsumerParams
		if err := c.BodyParser(&consumer); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		consumer.Id = c.Params("id")
		if err := EditConsumer(db, &consumer); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"message": "Consumer updated successfully"})
	})

	app.Delete("/consumers/:id", func(c *fiber.Ctx) error {
		consumerID := c.Params("id")
		if err := DeleteConsumer(db, consumerID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
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
}
