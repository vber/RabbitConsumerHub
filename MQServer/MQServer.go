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

	"go-rabbitmq-consumers/db"
	"go-rabbitmq-consumers/models"

	"github.com/streadway/amqp"
)

var ()

type RabbitMQServer struct {
	Connnection  *amqp.Connection
	ServerConfig *models.RabbitMQConfig
	RetryChan    chan error // 重试通道
	Connected    chan bool
	Stop         context.CancelFunc
	StopCtx      context.Context
	Consumer     *models.ConsumerParams
	DoError      ErrorHandler
	DoSuccess    SuccessHandler
}

type ErrorHandler func(queueData string, consumer *models.ConsumerParams)
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
		go mq.Connect(mq.Consumer.VHost)
	}
}

func NewRabbitMQServer(conf *models.RabbitMQConfig) *RabbitMQServer {
	if conf == nil {
		logger.E("NewRabbitMQServer", "RabbitMQConfig is nil")
		return nil
	}
	return &RabbitMQServer{
		ServerConfig: conf,
	}
}

func (mq *RabbitMQServer) Connect(vhost string) bool {
	var (
		err error
	)

	mq.StopCtx, mq.Stop = context.WithCancel(context.Background())

	mq.Connnection, err = amqp.DialConfig(fmt.Sprintf("amqp://%s:%s@%s:%d", mq.ServerConfig.User, mq.ServerConfig.Password, mq.ServerConfig.Host, mq.ServerConfig.Port), amqp.Config{
		Vhost: vhost,
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

func (mq *RabbitMQServer) validateCallbackResult(queuedata string, responseBody string, status_code int) {
	var (
		err         error
		cb_data     models.CallbackData
		bErrorFound bool
	)
	cb_data = models.CallbackData{}

	if err = json.Unmarshal([]byte(responseBody), &cb_data); err != nil {
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
		retryIntervals := []time.Duration{5 * time.Second, 1 * time.Minute, 5 * time.Minute}
		for _, interval := range retryIntervals {
			time.Sleep(interval)
			body, retryErr, retryStatusCode := utils.HttpRequest(utils.HTTP_POST, nil, mq.Consumer.Callback, queuedata)
			if retryErr == nil && retryStatusCode == 200 {
				var retryCbData models.CallbackData
				if json.Unmarshal([]byte(body), &retryCbData) == nil && retryCbData.ErrorCode == 0 {
					return // Retry successful
				}
			}
			responseBody = body           // Update responseBody with the latest retry response
			status_code = retryStatusCode // Update status_code with the latest retry status
		}

		// All retries failed, save to database
		if err := db.SaveFailedRequest(mq.Consumer.Callback, queuedata, responseBody, status_code); err != nil {
			logger.E("validateCallbackResult", "Failed to save failed request", err.Error())
			return
		}
	}
}

func (mq *RabbitMQServer) StartConsumer(params *models.ConsumerParams) error {
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

func (mq *RabbitMQServer) DeleteQueue() error {
	if mq.Connnection == nil {
		return errors.New("RabbitMQ Connection is nil")
	}

	ch, err := mq.Connnection.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	_, err = ch.QueueDelete(mq.Consumer.QueueName, false, false, false)
	if err != nil {
		return err
	}

	if mq.Consumer.DeathQueue.QueueName != "" {
		_, err = ch.QueueDelete(mq.Consumer.DeathQueue.QueueName, false, false, false)
	}

	return err
}
