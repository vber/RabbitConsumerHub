/*
	记录访问日志
*/

package logrecorder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vber/nacos/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	mongoDBClient     *mongo.Client
	MongoDBURI_Master string
	mq_log_collection *mongo.Collection
)

type MongoConfig struct {
	Master   string `json:"mongo_client_master"`
	Slave    string `json:"mongo_client_slave"`
	Port     int    `json:"mongo_port"`
	User     string `json:"mongo_auth"`
	Password string `json:"mongo_password"`
}

type LogData struct {
	MessageID            string             `bson:"message_id" json:"message_id"`
	ExchangeName         string             `bson:"exchange_name" json:"exchange_name"`
	QueueName            string             `bson:"queue_name" json:"queue_name"`
	RoutingKey           string             `bson:"routing_key" json:"routing_key"`
	QueueData            string             `bson:"queue_data" json:"queue_data"`
	RequestClient        string             `bson:"request_client" json:"request_client"`
	RequestClientVersion string             `bson:"request_client_version" json:"request_client_version"`
	Callback             string             `bson:"callback" json:"callback"`
	CallbackStatusCode   int                `bson:"callback_status_code" json:"callback_status_code"`
	CallbackData         string             `bson:"callback_data" json:"callback_data"`
	CallbackError        string             `bson:"callback_error,omitempty" json:"callback_error"`
	ReceiveTime          primitive.DateTime `bson:"receive_time" json:"receive_time"`
	ResponseTime         primitive.DateTime `bson:"response_time" json:"response_time"`
}

func getMongoDBURI() string {
	var (
		config string
		err    error
	)
	config, err = nacos.GetString("db.mongodb", "vinehoo.accounts", func(data *string, err error) {
		if err != nil {
			return
		}
		mongo_config := &MongoConfig{}
		if err = json.Unmarshal([]byte(*data), mongo_config); err == nil {
			MongoDBURI_Master = fmt.Sprintf("mongodb://%s:%s@%s:%d", mongo_config.User, mongo_config.Password, mongo_config.Master, mongo_config.Port)
		}
	})
	if err != nil || config == "" {
		panic(fmt.Errorf("failed to get mongodb config.%s", err.Error()))
	}

	mongo_config := &MongoConfig{}
	if err = json.Unmarshal([]byte(config), mongo_config); err != nil {
		panic(fmt.Errorf("failed to get mongodb config.%s", err.Error()))
	}

	return fmt.Sprintf("mongodb://%s:%s@%s:%d", mongo_config.User, mongo_config.Password, mongo_config.Master, mongo_config.Port)
}

func init() {
	MongoDBURI_Master = getMongoDBURI()
	if err := connect_to_mongodb(); err != nil {
		panic(fmt.Errorf("failed to connect to mongodb:%s", err.Error()))
	}
	checkConnection()

	mq_log_collection = mongoDBClient.Database("vinehoo_v3").Collection("rabbitmq_logs")
}

func connect_to_mongodb() error {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		err    error
	)

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoDBClient, err = mongo.Connect(ctx, options.Client().ApplyURI(MongoDBURI_Master))

	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}

func is_mongo_alive() bool {
	if mongoDBClient == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := mongoDBClient.Ping(ctx, readpref.Primary())
	return err == nil
}

func checkConnection() bool {
	if is_mongo_alive() == false {
		err := connect_to_mongodb()
		if err != nil {
			fmt.Println(err)
			return false
		}
		if is_mongo_alive() == false {
			fmt.Println("mongodb is not alive.")
			return false
		}
	}
	return true
}

func InsertLogDataToMongoDB(logData LogData) error {
	var (
		err    error
		ctx    context.Context
		cancel context.CancelFunc
	)
	if !checkConnection() {
		return errors.New("Connect to MongoDB failed")
	}

	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = mq_log_collection.InsertOne(ctx, logData)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}
