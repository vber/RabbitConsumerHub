package MQServer

import (
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

var (
	conn *amqp.Connection
)

func CreateDeathQueue(config *RabbitMQConfig, vhost string, v map[string]interface{}) (e error) {
	var (
		x_death_queue_name        string
		x_dead_letter_exchange    string
		x_dead_letter_routing_key string
		x_message_ttl             time.Duration
		bind_exchange             string
		bind_routing_key          string
		channel                   *amqp.Channel
		err                       error
	)

	defer func() {
		if _err := recover(); _err != nil {
			e = _err.(error)
		}
	}()

	if x_message_ttl, err = time.ParseDuration(v["x_message_ttl"].(string)); err != nil {
		panic(err)
	}

	x_death_queue_name = v["x_death_queue_name"].(string)
	x_dead_letter_exchange = v["x_dead_letter_exchange"].(string)
	x_dead_letter_routing_key = v["x_dead_letter_routing_key"].(string)
	bind_exchange = v["bind_exchange"].(string)
	bind_routing_key = v["bind_routing_key"].(string)

	if conn, err = amqp.DialConfig(fmt.Sprintf("amqp://%s:%s@%s:5672", config.User, config.Password, config.Host), amqp.Config{
		Vhost: vhost,
	}); err != nil {
		return err
	}

	defer conn.Close()

	if channel, err = conn.Channel(); err != nil {
		return err
	}

	defer channel.Close()

	_, err = channel.QueueDeclare(x_death_queue_name, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    x_dead_letter_exchange,
		"x-dead-letter-routing-key": x_dead_letter_routing_key,
		"x-message-ttl":             x_message_ttl.Milliseconds(),
		"durable":                   true,
	})

	if err != nil {
		return err
	}

	err = channel.QueueBind(x_death_queue_name, bind_routing_key, bind_exchange, false, nil)
	if err != nil {
		return err
	}

	return nil
}
