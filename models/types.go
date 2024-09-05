package models

type RabbitMQConfig struct {
	Host     string `json:"HOSTNAME"`
	Port     int    `json:"PORT"`
	User     string `json:"USERNAME"`
	Password string `json:"PASSWORD"`
}

type DeathQueueInfo struct {
	QueueName      string `json:"x_death_queue_name"`
	TTL            string `json:"x_message_ttl"`
	BindExchange   string `json:"bind_exchange"`
	BindRoutingKey string `json:"bind_routing_key"`
}

type ConsumerParams struct {
	Id               string         `json:"id"`
	Name             string         `json:"name"`
	Description      string         `json:"descripton"`
	AutoDecodeBase64 bool           `json:"auto_decode_base64"`
	Callback         string         `json:"callback"`
	ExchangeName     string         `json:"exchange_name"`
	RoutingKey       string         `json:"routing_key"`
	QueueName        string         `json:"queue_name"`
	VHost            string         `json:"vhost"`
	Status           string         `json:"status"`
	DingRobotToken   string         `json:"dingrobot_token"`
	RetryMode        string         `json:"retry_mode"`
	QueueCount       uint64         `json:"queue_count"`
	DeathQueue       DeathQueueInfo `json:"death_queue"`
	Qos              int            `json:"qos"`
}

type RabbitMQConsumers struct {
	Consumers []ConsumerParams `json:"consumers"`
}

// Add this to the existing types in models/types.go

type CallbackData struct {
	ErrorCode int64  `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}
