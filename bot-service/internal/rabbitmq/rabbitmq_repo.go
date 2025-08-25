package rabbitmq

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

type TaskNotify struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ChatID      int    `json:"chatID"`
}

const DelayedExchange = "delayed-exchange"
const NotifyTaskQueue = "notify-task-queue"
const NotifyKey = "notify"

func NewRabbitMQ(url string) *RabbitMQ {
	conn, err := amqp.Dial(url)
	if err != nil {
		logrus.Errorf("can`t connect to rabbitMQ, err%v", err)
		panic(err)
	}
	logrus.Info("Connected to rabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		logrus.Errorf("NewRabbitMQ, can`t create rabbitmq chan, err%v", err)
		panic(err)
	}
	rmq := &RabbitMQ{
		conn:    conn,
		channel: ch,
	}
	return rmq
}

func (r *RabbitMQ) DeclareQueue(exchangeName, queueName, routingKey string) {
	// Create queue
	q, err := r.channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logrus.Errorf("DeclareQueue, can`t create queue, err:%v", err)
		panic(err)
	}

	// Connect queue and exchange
	err = r.channel.QueueBind(
		q.Name,
		routingKey,
		exchangeName,
		false,
		nil,
	)
	if err != nil {
		logrus.Errorf("DeclareQueue, can`t connect queue and exchange, err:%v", err)
		panic(err)
	}
}

func (r *RabbitMQ) ConsumeChan(queueName string) <-chan amqp.Delivery {
	msgs, err := r.channel.Consume(
		queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logrus.Errorf("rabbitMQ, ConsumeChan, can`t consume from chan, err%v", err)
		panic(err)
	}
	return msgs
}

func (r *RabbitMQ) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}
