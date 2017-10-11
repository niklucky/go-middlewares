package middlewares

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

const (
	statusConnecting = "CONNECTING"
	statusConnected  = "CONNECTED"
)

/*
RabbitMQ - middleware for Rabbit MQ AMQP queue manager
Connects to Exchange and listens to Events
Send events to Exchange
*/
type RabbitMQ struct {
	Host
	State    string
	Conn     *amqp.Connection
	Channel  *amqp.Channel
	Exchange MQExchange
	handler  func([]byte) error
	m        sync.Mutex
	Debug    bool
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

// Connect - Connecting to Exchange
func (r *RabbitMQ) Connect() error {
	r.m.Lock()
	if r.State == statusConnecting {
		time.Sleep(1 * time.Second)
	}
	if r.State == statusConnecting {
		return nil
	}
	r.State = statusConnecting
	fmt.Println("[LOG][MQ] Connecting to: ", r.getAddressString())
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
	r.State = statusConnecting
	err = ch.ExchangeDeclare(
		r.Exchange.Name,
		r.Exchange.Type,
		r.Exchange.Durable,
		r.Exchange.AutoDeleted,
		false, // internal
		r.Exchange.NoWait,
		nil, // arguments
	)
	r.m.Unlock()
	if err != nil {
		fmt.Println("[ERROR][MQ] Error in ExchangeDeclare: ", err)
	}
	fmt.Println("[LOG][MQ] Connected to: ", r.getAddressString())
	return err
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
		err := r.Connect()
		if err != nil {
			return err
		}
	}
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if r.Debug {
		fmt.Println("[DEBUG] Message: ", string(body))
	}
	return r.Channel.Publish(
		r.Exchange.Name,
		r.Exchange.RoutingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		})
}

func (r *RabbitMQ) AddConsumer(h func([]byte) error) {
	r.handler = h
}

/*
Consume - declaring queue, binding to Exchange and starting to consume (listen) messages
*/
func (r *RabbitMQ) Consume() error {
	if r.isConnected() == false {
		err := r.Connect()
		if err != nil {
			return err
		}
	}
	q, err := r.Channel.QueueDeclare(
		r.Exchange.QueueName,
		false,
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
			var err error
			if r.handler != nil {
				err = r.handler(d.Body)
			}
			if err != nil {
				d.Ack(false)
			}
		}
	}()

	<-forever
	log.Println("Closing")
	r.Close()
	return nil
}

func (r *RabbitMQ) getAddressString() string {
	return "amqp://" + r.Host.User + ":" + r.Host.Password + "@" + r.Host.Host + ":" + strconv.Itoa(r.Host.Port)
}

func (r *RabbitMQ) isConnected() bool {
	return r.Conn != nil
}
