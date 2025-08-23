package rabbitmq

import (
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

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
	args := amqp.Table{
		"x-delayed-type": "direct",
	}

	// Create delayed exchange
	err := r.channel.ExchangeDeclare(
		exchangeName,
		"x-delayed-message",
		true,
		false,
		false,
		false,
		args,
	)
	if err != nil {
		logrus.Errorf("DeclareQueue, can`t create delayed exchange, err:%v", err)
		panic(err)
	}

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

func (r *RabbitMQ) Publish(exchangeName, queueName, rountingKey, body string, timer time.Time, date time.Time) error {
	// Prepare date
	targetTime := time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		timer.Hour(),
		timer.Minute(),
		0,
		0,
		time.Local,
	)

	delay := targetTime.Sub(time.Now())
	if delay <= 0 {
		err := fmt.Errorf("Publish rabbitMQ, time in the past")
		logrus.Error(err)
		return err
	}

	headers := amqp.Table{
		"x-delay": int64(delay.Milliseconds()),
	}

	// Publish
	err := r.channel.Publish(
		exchangeName,
		rountingKey,
		false,
		false,
		amqp.Publishing{
			Headers:     headers,
			ContentType: "text/plain",
			Body:        []byte(body),
		},
	)
	if err != nil {
		logrus.Errorf("Publish rabbitMQ, can`t publish, err:%v", err)
		return err
	}

	return nil
}

func (r *RabbitMQ) Close() {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		r.conn.Close()
	}
}

// 	date := time.Date(2025, 2, 26, 0, 0, 0, 0, time.Local)
// 	timer := time.Date(0, 0, 0, 15, 0, 0, 0, time.Local)
