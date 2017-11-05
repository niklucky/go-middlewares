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

type RabbitMQEvent int

const (
	RMQDisconnected RabbitMQEvent = iota
	RMQError
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
	Queue    amqp.Queue
	hData    func([]byte) error
	hEvent   func(RabbitMQEvent, interface{}) error
	Debug    bool
	sync.Mutex
}

// MQExchange - setting for MQ exchange
type MQExchange struct {
	Name            string
	Type            string
	RoutingKey      string `json:"routing_key"`
	Durable         bool
	AutoDeleted     bool `json:"auto_deleted"`
	NoWait          bool
	QueueName       string `json:"queue_name"`
	QueueDurable    bool   `json:"queue_durable"`
	QueueAutoDelete bool   `json:"queue_auto_delete"`
	QueueExclusive  bool   `json:"queue_exclusive"`
	C_AutoAck       bool   `json:"queue_auto_ack"`
	C_Exclusive     bool
}

// Connect - Connecting to Exchange
func (r *RabbitMQ) Connect() error {
	r.Lock()
	defer r.Unlock()
	if r.State == statusConnecting {
		time.Sleep(1 * time.Second)
	}
	if r.State == statusConnecting {
		return nil
	}
	r.State = statusConnecting
	fmt.Println("[LOG][MQ] Connecting to: ", r.getInfo())
	conn, err := amqp.Dial(r.getAddressString())
	if err != nil {
		logOnError(err, "Dial")
		r.State = ""
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

	if err != nil {
		fmt.Println("[ERROR][MQ] Error in ExchangeDeclare: ", err)
	}
	fmt.Println("[LOG][MQ] Connected to: ", r.getInfo())
	return err
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

/*
GetConnectedMQ - closing connections
*/
func GetConnectedMQ(host Host, ex MQExchange, hd func([]byte) error) (rmq RabbitMQ, err error) {
	rmq = RabbitMQ{
		Host:     host,
		Exchange: ex,
		hData:    hd,
	}

	for i := 0; i < max(1, host.Reconnect); i++ {
		err = rmq.Connect()
		if err != nil {
			logOnError(err, "GetConnectedMQ() error. Reconnecting...")
			if host.Reconnect > 0 {
				time.Sleep(time.Duration(max(1, host.Delay)) * time.Second)
			}
		} else {
			break
		}
	}

	if len(ex.QueueName) > 0 {
		rmq.Queue, err = rmq.QueueInit()
	}
	return rmq, err
}

/*
Close - closing connections
*/
func (r *RabbitMQ) Close() error {
	if r.Conn != nil {
		r.Conn.Close()
		r.Channel.Close()
	}
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
	r.hData = h
}

func (r *RabbitMQ) AddNotifyer(h func(RabbitMQEvent, interface{}) error) {
	r.hEvent = h
}

func (r *RabbitMQ) QueueInit() (q amqp.Queue, err error) {
	if r.isConnected() == false {
		err = r.Connect()
		if err != nil {
			return
		}
	}

	q, err = r.Channel.QueueDeclare(
		r.Exchange.QueueName,
		r.Exchange.QueueDurable,
		r.Exchange.QueueAutoDelete,
		r.Exchange.QueueExclusive,
		r.Exchange.NoWait,
		nil, // arguments
	)
	if err != nil {
		return
	}

	err = r.Channel.QueueBind(
		q.Name,                // queue name
		r.Exchange.RoutingKey, // routing key
		r.Exchange.Name,       // exchange
		false,
		nil)

	return
}

/*
Consume - declaring queue, binding to Exchange and starting to consume (listen) messages
*/
func (r *RabbitMQ) Consume() (err error) {
	r.Queue, err = r.QueueInit()
	if err != nil {
		return
	}

	msgs, err := r.Channel.Consume(
		r.Queue.Name,           // queue
		"",                     // consumer
		r.Exchange.C_AutoAck,   // auto-ack
		r.Exchange.C_Exclusive, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return
	}

	go func() {
		for d := range msgs {
			var err error
			if r.hData != nil {
				err = r.hData(d.Body)
			}
			if r.Exchange.C_AutoAck == false {
				if err == nil {
					d.Ack(false)
				}
			}
		}
		if r.hEvent != nil {
			r.hEvent(RMQDisconnected, nil)
		}
	}()

	log.Printf("Consuming %s ...", r.Queue.Name)
	return
}

func (r *RabbitMQ) getAddressString() string {
	return "amqp://" + r.Host.User + ":" + r.Host.Password + "@" + r.Host.Host + ":" + strconv.Itoa(r.Host.Port)
}
func (r *RabbitMQ) getInfo() string {
	return "amqp://" + r.Host.User + ":***@" + r.Host.Host + ":" + strconv.Itoa(r.Host.Port) + " | " + r.Exchange.Name
}

func (r *RabbitMQ) isConnected() bool {
	return r.Conn != nil
}
