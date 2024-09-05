package db

import (
	"database/sql"
	"fmt"
	"go-rabbitmq-consumers/logger"
	"go-rabbitmq-consumers/models"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(dbPath string) (*sql.DB, error) {
	const FUNCNAME = "InitDB"

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.E(FUNCNAME, "failed to open SQLite database.", err.Error())
		return nil, err
	}

	createTableSQLs := []string{
		`CREATE TABLE IF NOT EXISTS rabbitmq_config (
			id INTEGER PRIMARY KEY,
			host TEXT,
			port INTEGER,
			user TEXT,
			password TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS consumers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			status TEXT,
			queue_name TEXT,
			exchange_name TEXT,
			routing_key TEXT,
			vhost TEXT DEFAULT '/',
			death_queue_name TEXT DEFAULT '',
			death_queue_bind_exchange TEXT DEFAULT '',
			death_queue_bind_routing_key TEXT DEFAULT '',
			death_queue_ttl TEXT DEFAULT '',
			callback TEXT,
			retry_mode TEXT DEFAULT '',
			queue_count INTEGER DEFAULT 1
		);`,
		`CREATE TABLE IF NOT EXISTS retry_service_url (
			id INTEGER PRIMARY KEY,
			url TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS url_failed (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_url TEXT,
			request_data TEXT,
			response_code INTEGER,
			response_content TEXT,
			queue_name TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, sqlStmt := range createTableSQLs {
		_, err = DB.Exec(sqlStmt)
		if err != nil {
			logger.E(FUNCNAME, "failed to create table.", err.Error())
			return nil, err
		}
	}

	// Insert default data if tables are empty
	insertDefaultData(DB)

	return DB, nil
}

func insertDefaultData(db *sql.DB) {
	const FUNCNAME = "insertDefaultData"

	// Check if rabbitmq_config table is empty
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM rabbitmq_config").Scan(&count)
	if err != nil {
		logger.E(FUNCNAME, "failed to count rows in rabbitmq_config.", err.Error())
		return
	}
	if count == 0 {
		_, err = db.Exec(`INSERT INTO rabbitmq_config (id, host, port, user, password) 
                          VALUES (1, 'localhost', 5672, 'admin', 'password')`)
		if err != nil {
			logger.E(FUNCNAME, "failed to insert default RabbitMQ configuration.", err.Error())
		}
	}

	// Check if retry_service_url table is empty
	err = db.QueryRow("SELECT COUNT(*) FROM retry_service_url").Scan(&count)
	if err != nil {
		logger.E(FUNCNAME, "failed to count rows in retry_service_url.", err.Error())
		return
	}
	if count == 0 {
		_, err = db.Exec("INSERT INTO retry_service_url (id, url) VALUES (1, 'http://default-retry-service-url')")
		if err != nil {
			logger.E(FUNCNAME, "failed to insert default RetryServiceURL.", err.Error())
		}
	}
}

func FetchRabbitMQConfig(db *sql.DB) (*models.RabbitMQConfig, error) {
	const FUNCNAME = "FetchRabbitMQConfig"

	row := db.QueryRow("SELECT host, port, user, password FROM rabbitmq_config WHERE id = 1")
	var rabbitMQConf models.RabbitMQConfig
	err := row.Scan(&rabbitMQConf.Host, &rabbitMQConf.Port, &rabbitMQConf.User, &rabbitMQConf.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.E(FUNCNAME, "no RabbitMQ configuration found.")
			return nil, fmt.Errorf("no RabbitMQ configuration found")
		}
		logger.E(FUNCNAME, "failed to query RabbitMQ configuration from SQLite database.", err.Error())
		return nil, err
	}

	return &rabbitMQConf, nil
}

func FetchConsumersConfig(db *sql.DB) (*models.RabbitMQConsumers, error) {
	const FUNCNAME = "FetchConsumersConfig"

	rows, err := db.Query("SELECT id, name, status, queue_name, exchange_name, routing_key, vhost, death_queue_name, death_queue_bind_exchange, death_queue_bind_routing_key, death_queue_ttl, callback, retry_mode, queue_count FROM consumers")
	if err != nil {
		logger.E(FUNCNAME, "failed to query consumers from SQLite database.", err.Error())
		return nil, err
	}
	defer rows.Close()

	consumersConf := &models.RabbitMQConsumers{}
	for rows.Next() {
		var consumer models.ConsumerParams
		var deathQueueName, deathQueueBindExchange, deathQueueBindRoutingKey, retryMode sql.NullString
		var deathQueueTTL string
		err = rows.Scan(&consumer.Id, &consumer.Name, &consumer.Status, &consumer.QueueName, &consumer.ExchangeName, &consumer.RoutingKey, &consumer.VHost, &deathQueueName, &deathQueueBindExchange, &deathQueueBindRoutingKey, &deathQueueTTL, &consumer.Callback, &retryMode, &consumer.QueueCount)
		if err != nil {
			logger.E(FUNCNAME, "failed to scan consumer row.", err.Error())
			return nil, err
		}
		consumer.DeathQueue.QueueName = deathQueueName.String
		consumer.DeathQueue.BindExchange = deathQueueBindExchange.String
		consumer.DeathQueue.BindRoutingKey = deathQueueBindRoutingKey.String
		consumer.DeathQueue.TTL = deathQueueTTL // Adjusted to handle TEXT type
		consumer.RetryMode = retryMode.String
		consumersConf.Consumers = append(consumersConf.Consumers, consumer)
	}

	return consumersConf, nil
}

func FetchRetryServiceURL(db *sql.DB) (string, error) {
	const FUNCNAME = "FetchRetryServiceURL"

	row := db.QueryRow("SELECT url FROM retry_service_url WHERE id = 1")
	var retryServiceURL string
	err := row.Scan(&retryServiceURL)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.E(FUNCNAME, "no RetryServiceURL found.")
			return "", fmt.Errorf("no RetryServiceURL found")
		}
		logger.E(FUNCNAME, "failed to query RetryServiceURL from SQLite database.", err.Error())
		return "", err
	}

	return retryServiceURL, nil
}

func UpdateRabbitMQConfig(db *sql.DB, config models.RabbitMQConfig) error {
	const FUNCNAME = "UpdateRabbitMQConfig"

	// Log the values of Host and User before updating
	logger.I(FUNCNAME, fmt.Sprintf("Updating RabbitMQ config: Host=%s, User=%s", config.Host, config.User))

	_, err := db.Exec(`UPDATE rabbitmq_config SET host = ?, port = ?, user = ?, password = ? WHERE id = 1`,
		config.Host, config.Port, config.User, config.Password)
	if err != nil {
		logger.E(FUNCNAME, "failed to update RabbitMQ configuration.", err.Error())
		return err
	}

	return nil
}

func SaveFailedRequest(requestURL, requestData, responseBody string, statusCode int, queueName string) error {
	const FUNCNAME = "SaveFailedRequest"

	_, err := DB.Exec(`
		INSERT INTO url_failed (request_url, request_data, response_code, response_content, queue_name)
		VALUES (?, ?, ?, ?, ?, ?)
	`, requestURL, requestData, statusCode, responseBody, queueName)

	if err != nil {
		logger.E(FUNCNAME, "Failed to save failed request", err.Error())
		return err
	}

	return nil
}
