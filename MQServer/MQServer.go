package MQServer

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	logrecorder "go-rabbitmq-consumers/DBRecorder"
	"go-rabbitmq-consumers/logger"
	"go-rabbitmq-consumers/utils"
	"time"

	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ()

type RabbitMQConfig struct {
	Host      string `json:"HOSTNAME"`
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

type VinehooRabbitMQServer struct {
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

func (mq *VinehooRabbitMQServer) CheckConnection() {
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

func (mq *VinehooRabbitMQServer) ReConnect() {
	for retry := range mq.RetryChan {
		logger.E("ReConnect", retry.Error(), "3秒后重试!")
		<-time.After(3 * time.Second)
		logger.I("ReConnect", "Reconnecting ...")
		go mq.Connect()
	}
}

func NewVinehooRabbitMQServer(config *RabbitMQConfig) *VinehooRabbitMQServer {
	var (
		mq_server VinehooRabbitMQServer
	)

	if config == nil {
		panic(errors.New("rabbitMQConfig is nil"))
	}

	mq_server = VinehooRabbitMQServer{
		ServerConfig: config,
	}

	mq_server.RetryChan = make(chan error)
	mq_server.Connected = make(chan bool, 1)

	// mq_server.StopCtx, mq_server.Stop = context.WithCancel(context.Background())

	go mq_server.ReConnect()

	return &mq_server
}

func (mq *VinehooRabbitMQServer) Connect() bool {
	var (
		err error
	)

	mq.StopCtx, mq.Stop = context.WithCancel(context.Background())

	mq.Connnection, err = amqp.DialConfig(fmt.Sprintf("amqp://%s:%s@%s:5672", mq.ServerConfig.User, mq.ServerConfig.Password, mq.ServerConfig.Host), amqp.Config{
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

func (mq *VinehooRabbitMQServer) StopConsumer() {
	mq.Connnection.Close()
	mq.Stop()
}

func (mq *VinehooRabbitMQServer) sendToWeComRobot(statusCode int, body string) {
	const (
		FUNCNAME = "sendToWeComRobot"
	)
	var (
		err error
	)

	if mq.Consumer.DingRobotToken == "" {
		logger.I(FUNCNAME, "No Robot token set.")
		return
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=%s", mq.Consumer.DingRobotToken)

	postdata := map[string]interface{}{
		"markdown": map[string]string{
			"content": fmt.Sprintf("### 队列异常 \n #### **队列名称:** %s \n #### **队列描述:** %s \n #### **队列ID:** %s \n #### **Exchange:** %s \n #### **RoutingKey:** %s \n #### **重试模式:** %s \n #### **HTTPCode:** %d \n #### **回调地址:** %s \n #### **回调返回:** \n > %s ",
				mq.Consumer.Name, mq.Consumer.Description, mq.Consumer.Id, mq.Consumer.ExchangeName, mq.Consumer.RoutingKey, mq.Consumer.RetryMode, statusCode, mq.Consumer.Callback, body),
		},
		"msgtype": "markdown",
	}

	str_postdata, _ := json.Marshal(postdata)
	_, err, _ = utils.HttpRequest(utils.HTTP_POST, nil, url, string(str_postdata))

	if err != nil {
		logger.E(FUNCNAME, err.Error())
		return
	}
}

func (mq *VinehooRabbitMQServer) sendToDingtalkRobot(statusCode int, body string) {
	var (
		err  error
		data string
	)

	const (
		FUNCNAME = "sendToDingtalkRobot"
	)
	if mq.Consumer.DingRobotToken == "" {
		logger.I(FUNCNAME, "No Robot token set.")
		return
	}

	url := fmt.Sprintf("https://oapi.dingtalk.com/robot/send?access_token=%s", mq.Consumer.DingRobotToken)

	postdata := map[string]interface{}{
		"at": map[string]bool{
			"isAtAll": false,
		},
		"markdown": map[string]string{
			"title": fmt.Sprintf("队列报警-%s", mq.Consumer.Name),
			"text": fmt.Sprintf("### 队列异常 \n #### **队列名称:** %s \n #### **队列描述:** %s \n #### **队列ID:** %s \n #### **Exchange:** %s \n #### **RoutingKey:** %s \n #### **重试模式:** %s \n #### **HTTPCode:** %d \n #### **回调地址:** %s \n #### **回调返回:** \n > %s ",
				mq.Consumer.Name, mq.Consumer.Description, mq.Consumer.Id, mq.Consumer.ExchangeName, mq.Consumer.RoutingKey, mq.Consumer.RetryMode, statusCode, mq.Consumer.Callback, body),
		},
		"msgtype": "markdown",
	}

	str_postdata, _ := json.Marshal(postdata)
	data, err, _ = utils.HttpRequest(utils.HTTP_POST, nil, url, string(str_postdata))

	if err != nil {
		logger.E(FUNCNAME, err.Error())
		return
	}

	fmt.Println(data)
}

func (mq *VinehooRabbitMQServer) validateCallbackResult(queuedata string, data string, status_code int) {
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
		// 发送钉钉推送
		// mq.sendToDingtalkRobot(status_code, data)
		mq.sendToWeComRobot(status_code, data)

		if mq.Consumer.RetryMode != "" && mq.DoError != nil {
			// 重试机制启动
			mq.DoError(queuedata, mq.Consumer)
		}
	}
}

func (mq *VinehooRabbitMQServer) saveLogToMongoDB(logData logrecorder.LogData) {
	logrecorder.InsertLogDataToMongoDB(logData)
}

func (mq *VinehooRabbitMQServer) StartConsumer(params *ConsumerParams) error {
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

				receive_time := primitive.NewDateTimeFromTime(time.Now().Add(time.Hour * 8))
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

				_requestClient := "unknown"
				if data.Headers["vinehoo-client"] != nil {
					_requestClient = data.Headers["vinehoo-client"].(string)
				}
				_requestClientVersion := "unknown"
				if data.Headers["vinehoo-client-version"] != nil {
					_requestClientVersion = data.Headers["vinehoo-client-version"].(string)
				}
				// 队列日志写入mongodb
				logData := logrecorder.LogData{
					ExchangeName:         params.ExchangeName,
					QueueName:            params.QueueName,
					RoutingKey:           params.RoutingKey,
					MessageID:            data.MessageId,
					QueueData:            queue_data,
					RequestClient:        _requestClient,
					RequestClientVersion: _requestClientVersion,
					Callback:             params.Callback,
					CallbackData:         body,
					CallbackStatusCode:   statusCode,
					ReceiveTime:          receive_time,
					ResponseTime:         primitive.NewDateTimeFromTime(time.Now().Add(time.Hour * 8)),
				}
				if err != nil {
					logData.CallbackError = err.Error()
					logger.E("Callback failed:%s, url:%s", err.Error(), params.Callback)
				} else {
					logger.I("Callback", fmt.Sprintf("%s return:%s", params.Callback, body))
					go mq.validateCallbackResult(queue_data, body, statusCode)
				}

				ch.Ack(data.DeliveryTag, false)

				// 过滤
				if logData.Callback != "http://go-prometheus-api-log-consumer/logs/v3/prometheus" {
					go mq.saveLogToMongoDB(logData)
				}
			}
		}
	}()

	return nil
}
