package MQServer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go-rabbitmq-consumers/logger"
	"go-rabbitmq-consumers/utils"
	"time"

	"github.com/streadway/amqp"
)

var ()

type RabbitMQConfig struct {
	Host      string `json:"HOSTNAME"`
	Port      int    `json:"PORT"` // Added Port field
	User      string `json:"USERNAME"`
	Password  string `json:"PASSWORD"`
	Heartbeat uint64 `json:"HEARTBEAT"`
	FrameSize int    `json:"FRAMEMAX"`
	Vhost     string `json:"VHOST"`
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
	Status           string         `json:"status"`
	DingRobotToken   string         `json:"dingrobot_token"`
	RetryMode        string         `json:"retry_mode"`
	QueueCount       uint64         `json:"queue_count"`
	DeathQueue       DeathQueueInfo `json:"death_queue"`
	Qos              int            `json:"qos"`
}

type CallbackData struct {
	ErrorCode int64  `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

type RabbitMQConsumers struct {
	Consumers []ConsumerParams `json:"consumers"`
}

type RabbitMQServer struct {
	Connnection  *amqp.Connection
	ServerConfig *RabbitMQConfig
	RetryChan    chan error // 重试通道
	Connected    chan bool
	Stop         context.CancelFunc
	StopCtx      context.Context
	Consumer     *ConsumerParams
	DoError      ErrorHandler
	DoSuccess    SuccessHandler
}

type ErrorHandler func(queueData string, consumer *ConsumerParams)
type SuccessHandler func(retry_id string)

func init() {

}

func (mq *RabbitMQServer) CheckConnection() {
	for {
		select {
		case <-mq.StopCtx.Done():
			return
		default:
			if mq.Connnection.IsClosed() {
				mq.RetryChan <- fmt.Errorf("queue %s has been disconnected", mq.Consumer.Name)
				return
			}
		}
		time.Sleep(3 * time.Second)
	}
}

func (mq *RabbitMQServer) ReConnect() {
	for retry := range mq.RetryChan {
		logger.E("ReConnect", retry.Error(), "3秒后重试!")
		<-time.After(3 * time.Second)
		logger.I("ReConnect", "Reconnecting ...")
		go mq.Connect()
	}
}

func NewRabbitMQServer(config *RabbitMQConfig) *RabbitMQServer {
	var (
		mq_server RabbitMQServer
	)

	if config == nil {
		panic(errors.New("rabbitMQConfig is nil"))
	}

	mq_server = RabbitMQServer{
		ServerConfig: config,
	}

	mq_server.RetryChan = make(chan error)
	mq_server.Connected = make(chan bool, 1)

	go mq_server.ReConnect()

	return &mq_server
}

func (mq *RabbitMQServer) Connect() bool {
	var (
		err error
	)

	mq.StopCtx, mq.Stop = context.WithCancel(context.Background())

	mq.Connnection, err = amqp.DialConfig(fmt.Sprintf("amqp://%s:%s@%s:%d", mq.ServerConfig.User, mq.ServerConfig.Password, mq.ServerConfig.Host, mq.ServerConfig.Port), amqp.Config{
		Vhost:     mq.ServerConfig.Vhost,
		FrameSize: mq.ServerConfig.FrameSize,
		Heartbeat: time.Duration(mq.ServerConfig.Heartbeat),
	})

	if err != nil {
		mq.RetryChan <- err

		return false
	}

	go mq.CheckConnection()

	if mq.Consumer != nil {
		mq.StartConsumer(mq.Consumer)
	}

	return true
}

func (mq *RabbitMQServer) StopConsumer() {
	mq.Connnection.Close()
	mq.Stop()
}

func (mq *RabbitMQServer) validateCallbackResult(queuedata string, data string, status_code int) {
	var (
		err         error
		cb_data     CallbackData
		bErrorFound bool
	)
	cb_data = CallbackData{}

	if err = json.Unmarshal([]byte(data), &cb_data); err != nil {
		logger.E("validateCallbackResult", err.Error())
	}

	if status_code != 200 {
		bErrorFound = true
	} else {
		if cb_data.ErrorCode == 0 && err == nil {
			return
		} else {
			bErrorFound = true
		}
	}

	if bErrorFound {
		if mq.Consumer.RetryMode != "" && mq.DoError != nil {
			// 重试机制启动
			mq.DoError(queuedata, mq.Consumer)
		}
	}
}

func (mq *RabbitMQServer) StartConsumer(params *ConsumerParams) error {
	var (
		ch  *amqp.Channel
		q   amqp.Queue
		err error
	)
	if mq.Connnection == nil {
		return errors.New("RabbitMQ Connection is nil")
	}

	ch, err = mq.Connnection.Channel()
	if err != nil {
		return err
	}
	if params.Qos == 0 {
		ch.Qos(1, 0, false)
	} else {
		ch.Qos(params.Qos, 0, false)
	}

	if err = ch.ExchangeDeclare(params.ExchangeName, "topic", true, false, false, false, nil); err != nil {
		return err
	}

	q, err = ch.QueueDeclare(params.QueueName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	err = ch.QueueBind(params.QueueName, params.RoutingKey, params.ExchangeName, false, nil)
	if err != nil {
		return err
	}

	msg, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	mq.Consumer = params

	go func() {
		var (
			queue_data string
			body       string
			tmp_data   []byte
			err        error
			statusCode int
		)
		defer func() {
			logger.I("Close", "queuename:", mq.Consumer.QueueName)
			if err = ch.Close(); err != nil {
				logger.E("Close", err.Error())
			}
			if err = mq.Connnection.Close(); err != nil {
				logger.E("Close", err.Error())
			}
		}()

		for {
			select {
			case <-mq.StopCtx.Done():
				return
			case data := <-msg:
				// fmt.Println("msg:", string(data.Body), "callback:", params.Callback, data.Headers["vinehoo-retry-id"])
				if mq.Connnection.IsClosed() {
					return
				}

				// receive_time := primitive.NewDateTimeFromTime(time.Now().Add(time.Hour * 8))
				queue_data = string(data.Body)
				if params.AutoDecodeBase64 {
					tmp_data, err = base64.StdEncoding.DecodeString(queue_data)
					queue_data = string(tmp_data)
					if err != nil {
						logger.I("Decodebase64", err.Error())
						fmt.Println(err)
					}
				}

				logger.I("Consumer", fmt.Sprintf("id:%s, queue_name:%s, callback:%s, data:%s", params.Id, params.Name, params.Callback, queue_data))

				body, err, statusCode = utils.HttpRequest(utils.HTTP_POST, nil, params.Callback, queue_data)

				logger.I("Callback", fmt.Sprintf("%s return:%s", params.Callback, body))
				go mq.validateCallbackResult(queue_data, body, statusCode)

				ch.Ack(data.DeliveryTag, false)
			}
		}
	}()

	return nil
}
