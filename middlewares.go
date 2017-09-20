package middlewares

import (
	"log"
)

/*
NewRabbitMQ - RqbbitMQ middleware constructor
*/
func NewRabbitMQ(host Host, ex MQExchange) RabbitMQ {
	return RabbitMQ{
		Host:     host,
		Exchange: ex,
	}
}

/*
NewWebsocketClient - constructor for WebsocketClient
*/
func NewWebsocketClient(host Host) WebsocketClient {
	return WebsocketClient{
		Host: host,
	}
}

// Host - struct for host params config for HTTP/WS connections
type Host struct {
	Host     string
	Port     int
	User     string
	Password string
	Path     string
	Data     interface{}
}

func logOnError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}
