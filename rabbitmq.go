package middlewares

import (
	"encoding/json"
	"strconv"

	"github.com/streadway/amqp"
)

/*
RabbitMQ - middleware for Rabbit MQ AMQP queue manager
Connects to Exchange and listens to Events
Send events to Exchange
*/
type RabbitMQ struct {
	Host
	Conn     *amqp.Connection
	Channel  *amqp.Channel
	Exchange MQExchange
}

// MQExchange - setting for MQ exchange
type MQExchange struct {
	Name        string
	Type        string
	RoutingKey  string
	QueueName   string
	Durable     bool
	AutoDeleted bool
	NoWait      bool
}

// Connecting to Exchange
func (r *RabbitMQ) connect() error {
	conn, err := amqp.Dial(r.getAddressString())
	if err != nil {
		logOnError(err, "Dial")
		return err
	}
	r.Conn = conn
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	r.Channel = ch
	return ch.ExchangeDeclare(
		r.Exchange.Name,
		r.Exchange.Type,
		r.Exchange.Durable,
		r.Exchange.AutoDeleted,
		false, // internal
		r.Exchange.NoWait,
		nil, // arguments
	)
}

/*
Close - closing connections
*/
func (r *RabbitMQ) Close() error {
	r.Conn.Close()
	r.Channel.Close()
	return nil
}

/*
Publish — publishing message to RabbitMQ exchange
*/
func (r *RabbitMQ) Publish(data interface{}) error {
	if r.isConnected() == false {
		err := r.connect()
		if err != nil {
			return err
		}
	}
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.Channel.Publish(
		r.Exchange.Name,
		r.Exchange.RoutingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
}

/*
Consume - declaring queue, binding to Exchange and starting to consume (listen) messages
*/
func (r *RabbitMQ) Consume(handler func(interface{})) error {
	if r.isConnected() == false {
		err := r.connect()
		if err != nil {
			return err
		}
	}
	q, err := r.Channel.QueueDeclare(
		r.Exchange.QueueName,
		r.Exchange.Durable,
		false, // delete when usused
		true,  // exclusive
		r.Exchange.NoWait,
		nil, // arguments
	)

	err = r.Channel.QueueBind(
		q.Name,                // queue name
		r.Exchange.RoutingKey, // routing key
		r.Exchange.Name,       // exchange
		false,
		nil)
	if err != nil {
		return err
	}

	msgs, err := r.Channel.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return err
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			handler(d.Body)
			d.Ack(false)
		}
	}()

	<-forever
	r.Close()
	return nil
}

func (r *RabbitMQ) getAddressString() string {
	return "amqp://" + r.Host.User + ":" + r.Host.Password + "@" + r.Host.Host + ":" + strconv.Itoa(r.Host.Port)
}

func (r *RabbitMQ) isConnected() bool {
	return r.Conn != nil
}
